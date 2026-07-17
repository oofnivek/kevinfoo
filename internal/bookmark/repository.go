package bookmark

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("bookmark not found")

type Repository interface {
	List(ctx context.Context, query string) ([]Bookmark, error)
	GetByID(ctx context.Context, id string) (Bookmark, error)
	Create(ctx context.Context, b Bookmark) (Bookmark, error)
	Update(ctx context.Context, b Bookmark) (Bookmark, error)
	Delete(ctx context.Context, id string) error
}
