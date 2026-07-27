// Package loginlog records login attempts for auditing.
package loginlog

import (
	"context"
	"time"
)

// Attempt represents a single login attempt.
type Attempt struct {
	Username  string
	IP        string
	UserAgent string
	Success   bool
	Reason    string
	CreatedAt time.Time
}

// Logger records login attempts.
type Logger interface {
	Log(ctx context.Context, a Attempt) error
}
