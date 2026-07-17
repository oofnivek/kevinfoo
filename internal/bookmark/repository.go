package bookmark

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("bookmark not found")

type Repository interface {
	List(ctx context.Context, query string) ([]Bookmark, error)
	GetByID(ctx context.Context, id int64) (Bookmark, error)
	Create(ctx context.Context, b Bookmark) (Bookmark, error)
	Update(ctx context.Context, b Bookmark) (Bookmark, error)
	Delete(ctx context.Context, id int64) error
}

type sqliteRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) List(ctx context.Context, query string) ([]Bookmark, error) {
	var rows *sql.Rows
	var err error

	if query == "" {
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, title, url, description, tags, created_at, updated_at
			FROM bookmarks
			ORDER BY created_at DESC`)
	} else {
		like := "%" + query + "%"
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, title, url, description, tags, created_at, updated_at
			FROM bookmarks
			WHERE title LIKE ? OR url LIKE ? OR description LIKE ? OR tags LIKE ?
			ORDER BY created_at DESC`, like, like, like, like)
	}
	if err != nil {
		return nil, fmt.Errorf("list bookmarks: %w", err)
	}
	defer rows.Close()

	var bookmarks []Bookmark
	for rows.Next() {
		var b Bookmark
		if err := rows.Scan(&b.ID, &b.Title, &b.URL, &b.Description, &b.Tags, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan bookmark: %w", err)
		}
		bookmarks = append(bookmarks, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list bookmarks: %w", err)
	}

	return bookmarks, nil
}

func (r *sqliteRepository) GetByID(ctx context.Context, id int64) (Bookmark, error) {
	var b Bookmark
	err := r.db.QueryRowContext(ctx, `
		SELECT id, title, url, description, tags, created_at, updated_at
		FROM bookmarks WHERE id = ?`, id,
	).Scan(&b.ID, &b.Title, &b.URL, &b.Description, &b.Tags, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Bookmark{}, ErrNotFound
	}
	if err != nil {
		return Bookmark{}, fmt.Errorf("get bookmark %d: %w", id, err)
	}
	return b, nil
}

func (r *sqliteRepository) Create(ctx context.Context, b Bookmark) (Bookmark, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO bookmarks (title, url, description, tags)
		VALUES (?, ?, ?, ?)`, b.Title, b.URL, b.Description, b.Tags)
	if err != nil {
		return Bookmark{}, fmt.Errorf("create bookmark: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return Bookmark{}, fmt.Errorf("create bookmark: %w", err)
	}

	return r.GetByID(ctx, id)
}

func (r *sqliteRepository) Update(ctx context.Context, b Bookmark) (Bookmark, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE bookmarks
		SET title = ?, url = ?, description = ?, tags = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, b.Title, b.URL, b.Description, b.Tags, b.ID)
	if err != nil {
		return Bookmark{}, fmt.Errorf("update bookmark %d: %w", b.ID, err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return Bookmark{}, fmt.Errorf("update bookmark %d: %w", b.ID, err)
	}
	if n == 0 {
		return Bookmark{}, ErrNotFound
	}

	return r.GetByID(ctx, b.ID)
}

func (r *sqliteRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM bookmarks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete bookmark %d: %w", id, err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete bookmark %d: %w", id, err)
	}
	if n == 0 {
		return ErrNotFound
	}

	return nil
}
