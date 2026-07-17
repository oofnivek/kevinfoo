package auth

import (
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Renderer renders a named template to w.
type Renderer interface {
	Render(w io.Writer, name string, data any) error
}

type Handler struct {
	session         *Session
	username        string
	password        string
	recaptchaSite   string
	recaptchaSecret string
	httpClient      *http.Client
	render          func(w http.ResponseWriter, name string, data any)
}

func NewHandler(session *Session, username, password, recaptchaSiteKey, recaptchaSecretKey string, r Renderer, logger *slog.Logger) *Handler {
	return &Handler{
		session:         session,
		username:        username,
		password:        password,
		recaptchaSite:   recaptchaSiteKey,
		recaptchaSecret: recaptchaSecretKey,
		httpClient:      &http.Client{Timeout: 10 * time.Second},
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
		"Next":          safeNext(r.URL.Query().Get("next")),
		"RecaptchaSite": h.recaptchaSite,
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

	if h.recaptchaSecret != "" && !h.verifyRecaptcha(r.PostForm.Get("g-recaptcha-response"), r.RemoteAddr) {
		h.render(w, "login", map[string]any{
			"Next":          next,
			"RecaptchaSite": h.recaptchaSite,
			"Error":         "Please complete the reCAPTCHA.",
		})
		return
	}

	if !CheckCredentials(username, password, h.username, h.password) {
		h.render(w, "login", map[string]any{
			"Next":          next,
			"RecaptchaSite": h.recaptchaSite,
			"Error":         "Invalid username or password.",
		})
		return
	}

	h.session.IssueCookie(w, r)
	http.Redirect(w, r, next, http.StatusSeeOther)
}

// verifyRecaptcha checks a reCAPTCHA v2 response token against Google's
// siteverify endpoint.
func (h *Handler) verifyRecaptcha(token, remoteAddr string) bool {
	if token == "" {
		return false
	}

	form := url.Values{
		"secret":   {h.recaptchaSecret},
		"response": {token},
	}
	if ip, _, err := net.SplitHostPort(remoteAddr); err == nil {
		form.Set("remoteip", ip)
	}

	resp, err := h.httpClient.PostForm("https://www.google.com/recaptcha/api/siteverify", form)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false
	}

	return result.Success
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	h.session.ClearCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// safeNext only allows same-site relative redirect targets, guarding
// against the "next" parameter being used as an open redirect. Backslashes
// are rejected outright: browsers treat "\" the same as "/" for http(s)
// URLs (per the WHATWG URL spec), so "/\evil.com" resolves the same way
// "//evil.com" does even though it doesn't look protocol-relative here.
func safeNext(next string) string {
	const fallback = "/bookmarks"

	if next == "" || strings.ContainsRune(next, '\\') {
		return fallback
	}

	u, err := url.Parse(next)
	if err != nil || u.Host != "" || u.Scheme != "" || u.Opaque != "" {
		return fallback
	}
	if !strings.HasPrefix(u.Path, "/") || strings.HasPrefix(u.Path, "//") {
		return fallback
	}

	return next
}
