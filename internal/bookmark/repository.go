package bookmark

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("bookmark not found")

type Repository interface {
	// List returns bookmarks matching query. sort is "asc" or "desc" to
	// order by title, or "" for the default (most recently created first).
	List(ctx context.Context, query, sort string) ([]Bookmark, error)
	GetByID(ctx context.Context, id string) (Bookmark, error)
	Create(ctx context.Context, b Bookmark) (Bookmark, error)
	Update(ctx context.Context, b Bookmark) (Bookmark, error)
	Delete(ctx context.Context, id string) error
}
