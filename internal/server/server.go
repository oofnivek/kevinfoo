// Package server assembles the HTTP routes for the application.
package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	"bookmarks/internal/bookmark"
)

// Renderer renders a named template to w.
type Renderer interface {
	Render(w io.Writer, name string, data any) error
}

func NewMux(bookmarks *bookmark.Handler, renderer Renderer, ping func(context.Context) error, staticDir string, logger *slog.Logger) *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	mux.HandleFunc("GET /healthz", healthCheck(ping))

	mux.HandleFunc("GET /{$}", homePage(renderer, logger))

	mux.HandleFunc("GET /bookmarks", bookmarks.Page)
	mux.HandleFunc("GET /bookmarks/list", bookmarks.List)
	mux.HandleFunc("POST /bookmarks", bookmarks.Create)
	mux.HandleFunc("GET /bookmarks/new", bookmarks.NewForm)
	mux.HandleFunc("GET /bookmarks/cancel", bookmarks.CancelForm)
	mux.HandleFunc("GET /bookmarks/{id}", bookmarks.Row)
	mux.HandleFunc("GET /bookmarks/{id}/edit", bookmarks.EditForm)
	mux.HandleFunc("PUT /bookmarks/{id}", bookmarks.Update)
	mux.HandleFunc("DELETE /bookmarks/{id}", bookmarks.Delete)

	return mux
}

func homePage(renderer Renderer, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := renderer.Render(w, "home", nil); err != nil {
			logger.Error("render template", "template", "home", "error", err)
		}
	}
}

func healthCheck(ping func(context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := ping(r.Context()); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}
}
