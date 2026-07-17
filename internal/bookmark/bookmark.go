// Package bookmark implements storage, business logic, and HTTP handlers
// for managing bookmarks.
package bookmark

import "time"

type Bookmark struct {
	ID          string
	Title       string
	URL         string
	Description string
	Tags        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
