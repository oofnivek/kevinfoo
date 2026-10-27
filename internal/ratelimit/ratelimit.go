// Package ratelimit implements an in-memory, per-client request rate limit.
package ratelimit

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"bookmarks/internal/clientip"
)

// Limiter allows one request per interval for each key. State lives in
// memory, so it is per-process and resets on restart.
type Limiter struct {
	interval time.Duration

	mu        sync.Mutex
	last      map[string]time.Time
	lastSweep time.Time
}

func New(interval time.Duration) *Limiter {
	return &Limiter{
		interval:  interval,
		last:      make(map[string]time.Time),
		lastSweep: time.Now(),
	}
}

// Allow reports whether a request for key may proceed, recording it if so.
func (l *Limiter) Allow(key string) bool {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	l.sweep(now)

	if t, ok := l.last[key]; ok && now.Sub(t) < l.interval {
		return false
	}
	l.last[key] = now
	return true
}

// sweep drops keys whose interval has passed, so the map doesn't grow
// without bound. It runs at most once a minute.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < time.Minute {
		return
	}
	for key, t := range l.last {
		if now.Sub(t) >= l.interval {
			delete(l.last, key)
		}
	}
	l.lastSweep = now
}

// Middleware rejects requests with 429 Too Many Requests when the client IP
// has already made a request to the same path within the interval.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.Allow(clientip.FromRequest(r) + " " + r.URL.Path) {
			w.Header().Set("Retry-After", strconv.Itoa(int(l.interval.Round(time.Second).Seconds())))
			http.Error(w, "too many requests, try again in a second", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
