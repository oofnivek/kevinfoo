// Package auth implements a minimal signed-cookie session for gating access
// to the app behind a single username/password stored in configuration.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	cookieName = "session"
	sessionTTL = 7 * 24 * time.Hour
)

type Session struct {
	Secret string
}

func New(secret string) *Session {
	return &Session{Secret: secret}
}

// IssueCookie sets a signed session cookie on the response.
func (s *Session) IssueCookie(w http.ResponseWriter, r *http.Request) {
	expiry := time.Now().Add(sessionTTL).Unix()
	value := s.sign(expiry)

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(expiry, 0),
	})
}

// isHTTPS reports whether the original client request used HTTPS. r.TLS is
// checked for direct TLS termination, but behind a reverse proxy (Render,
// most PaaS) TLS is terminated at the edge and the app only sees plain HTTP,
// so the proxy's X-Forwarded-Proto header is also trusted.
func isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// ClearCookie removes the session cookie.
func (s *Session) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

// Valid reports whether the request carries a valid, unexpired session cookie.
func (s *Session) Valid(r *http.Request) bool {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return false
	}

	expiry, ok := s.verify(c.Value)
	if !ok {
		return false
	}

	return time.Now().Unix() < expiry
}

func (s *Session) sign(expiry int64) string {
	payload := strconv.FormatInt(expiry, 10)
	mac := hmac.New(sha256.New, []byte(s.Secret))
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payload + "." + sig
}

func (s *Session) verify(cookieValue string) (int64, bool) {
	parts := strings.SplitN(cookieValue, ".", 2)
	if len(parts) != 2 {
		return 0, false
	}
	payload, sig := parts[0], parts[1]

	mac := hmac.New(sha256.New, []byte(s.Secret))
	mac.Write([]byte(payload))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if subtle.ConstantTimeCompare([]byte(sig), []byte(expectedSig)) != 1 {
		return 0, false
	}

	expiry, err := strconv.ParseInt(payload, 10, 64)
	if err != nil {
		return 0, false
	}

	return expiry, true
}

// CheckCredentials compares the given username/password against the
// configured ones using constant-time comparison.
func CheckCredentials(username, password, wantUsername, wantPassword string) bool {
	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(wantUsername)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(password), []byte(wantPassword)) == 1
	return userOK && passOK
}

// Middleware redirects unauthenticated requests to /login, preserving the
// originally requested path as a "next" query parameter.
func (s *Session) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.Valid(r) {
			next.ServeHTTP(w, r)
			return
		}

		loginURL := fmt.Sprintf("/login?next=%s", r.URL.Path)
		http.Redirect(w, r, loginURL, http.StatusSeeOther)
	})
}
