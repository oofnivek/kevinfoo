package auth

import (
	"io"
	"log/slog"
	"net/http"
)

// Renderer renders a named template to w.
type Renderer interface {
	Render(w io.Writer, name string, data any) error
}

type Handler struct {
	session  *Session
	username string
	password string
	render   func(w http.ResponseWriter, name string, data any)
}

func NewHandler(session *Session, username, password string, r Renderer, logger *slog.Logger) *Handler {
	return &Handler{
		session:  session,
		username: username,
		password: password,
		render: func(w http.ResponseWriter, name string, data any) {
			if err := r.Render(w, name, data); err != nil {
				logger.Error("render template", "template", name, "error", err)
			}
		},
	}
}

func (h *Handler) LoginForm(w http.ResponseWriter, r *http.Request) {
	if h.session.Valid(r) {
		http.Redirect(w, r, "/bookmarks", http.StatusSeeOther)
		return
	}

	h.render(w, "login", map[string]any{
		"Next": safeNext(r.URL.Query().Get("next")),
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	username := r.PostForm.Get("username")
	password := r.PostForm.Get("password")
	next := safeNext(r.PostForm.Get("next"))

	if !CheckCredentials(username, password, h.username, h.password) {
		h.render(w, "login", map[string]any{
			"Next":  next,
			"Error": "Invalid username or password.",
		})
		return
	}

	h.session.IssueCookie(w, r)
	http.Redirect(w, r, next, http.StatusSeeOther)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	h.session.ClearCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// safeNext only allows same-site relative redirect targets, guarding
// against the "next" parameter being used as an open redirect.
func safeNext(next string) string {
	if next == "" || next[0] != '/' || len(next) > 1 && next[1] == '/' {
		return "/bookmarks"
	}
	return next
}
