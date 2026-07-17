package bookmark

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var ErrValidation = errors.New("validation failed")

type Service interface {
	List(ctx context.Context, query string) ([]Bookmark, error)
	Get(ctx context.Context, id int64) (Bookmark, error)
	Create(ctx context.Context, title, rawURL, description, tags string) (Bookmark, error)
	Update(ctx context.Context, id int64, title, rawURL, description, tags string) (Bookmark, error)
	Delete(ctx context.Context, id int64) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) List(ctx context.Context, query string) ([]Bookmark, error) {
	return s.repo.List(ctx, strings.TrimSpace(query))
}

func (s *service) Get(ctx context.Context, id int64) (Bookmark, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) Create(ctx context.Context, title, rawURL, description, tags string) (Bookmark, error) {
	b, err := normalize(title, rawURL, description, tags)
	if err != nil {
		return Bookmark{}, err
	}
	return s.repo.Create(ctx, b)
}

func (s *service) Update(ctx context.Context, id int64, title, rawURL, description, tags string) (Bookmark, error) {
	b, err := normalize(title, rawURL, description, tags)
	if err != nil {
		return Bookmark{}, err
	}
	b.ID = id
	return s.repo.Update(ctx, b)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func normalize(title, rawURL, description, tags string) (Bookmark, error) {
	title = strings.TrimSpace(title)
	rawURL = strings.TrimSpace(rawURL)

	if title == "" {
		return Bookmark{}, fmt.Errorf("%w: title is required", ErrValidation)
	}

	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return Bookmark{}, fmt.Errorf("%w: url must be a valid http(s) URL", ErrValidation)
	}

	return Bookmark{
		Title:       title,
		URL:         rawURL,
		Description: strings.TrimSpace(description),
		Tags:        normalizeTags(tags),
	}, nil
}

func normalizeTags(tags string) string {
	parts := strings.Split(tags, ",")
	cleaned := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	return strings.Join(cleaned, ", ")
}
