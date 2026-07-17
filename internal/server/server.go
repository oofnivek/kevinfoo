// Package server assembles the HTTP routes for the application.
package server

import (
	"net/http"

	"bookmarks/internal/bookmark"
)

func NewMux(bookmarks *bookmark.Handler, staticDir string) *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	mux.HandleFunc("GET /{$}", bookmarks.Index)

	mux.HandleFunc("GET /bookmarks", bookmarks.List)
	mux.HandleFunc("POST /bookmarks", bookmarks.Create)
	mux.HandleFunc("GET /bookmarks/new", bookmarks.NewForm)
	mux.HandleFunc("GET /bookmarks/cancel", bookmarks.CancelForm)
	mux.HandleFunc("GET /bookmarks/{id}", bookmarks.Row)
	mux.HandleFunc("GET /bookmarks/{id}/edit", bookmarks.EditForm)
	mux.HandleFunc("PUT /bookmarks/{id}", bookmarks.Update)
	mux.HandleFunc("DELETE /bookmarks/{id}", bookmarks.Delete)

	return mux
}
