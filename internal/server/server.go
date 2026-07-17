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

// AuthHandler serves the login/logout routes.
type AuthHandler interface {
	LoginForm(w http.ResponseWriter, r *http.Request)
	Login(w http.ResponseWriter, r *http.Request)
	Logout(w http.ResponseWriter, r *http.Request)
}

func NewMux(bookmarks *bookmark.Handler, auth AuthHandler, protect func(http.Handler) http.Handler, authed func(*http.Request) bool, renderer Renderer, ping func(context.Context) error, staticDir string, logger *slog.Logger) *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	mux.HandleFunc("GET /healthz", healthCheck(ping))

	mux.HandleFunc("GET /login", auth.LoginForm)
	mux.HandleFunc("POST /login", auth.Login)
	mux.HandleFunc("POST /logout", auth.Logout)

	mux.HandleFunc("GET /{$}", homePage(renderer, logger, authed))
	mux.HandleFunc("GET /jwt-decoder", toolPage(renderer, logger, "jwt-decoder"))

	mux.Handle("GET /bookmarks", protect(http.HandlerFunc(bookmarks.Page)))
	mux.Handle("GET /bookmarks/list", protect(http.HandlerFunc(bookmarks.List)))
	mux.Handle("POST /bookmarks", protect(http.HandlerFunc(bookmarks.Create)))
	mux.Handle("GET /bookmarks/new", protect(http.HandlerFunc(bookmarks.NewForm)))
	mux.Handle("GET /bookmarks/cancel", protect(http.HandlerFunc(bookmarks.CancelForm)))
	mux.Handle("GET /bookmarks/{id}", protect(http.HandlerFunc(bookmarks.Row)))
	mux.Handle("GET /bookmarks/{id}/edit", protect(http.HandlerFunc(bookmarks.EditForm)))
	mux.Handle("PUT /bookmarks/{id}", protect(http.HandlerFunc(bookmarks.Update)))
	mux.Handle("DELETE /bookmarks/{id}", protect(http.HandlerFunc(bookmarks.Delete)))

	return mux
}

func homePage(renderer Renderer, logger *slog.Logger, authed func(*http.Request) bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{"LoggedIn": authed(r)}
		if err := renderer.Render(w, "home", data); err != nil {
			logger.Error("render template", "template", "home", "error", err)
		}
	}
}

func toolPage(renderer Renderer, logger *slog.Logger, name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := renderer.Render(w, name, nil); err != nil {
			logger.Error("render template", "template", name, "error", err)
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
