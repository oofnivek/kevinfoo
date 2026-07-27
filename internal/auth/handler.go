package auth

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"bookmarks/internal/loginlog"
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
	attempts        loginlog.Logger
	logger          *slog.Logger
	render          func(w http.ResponseWriter, name string, data any)
}

func NewHandler(session *Session, username, password, recaptchaSiteKey, recaptchaSecretKey string, attempts loginlog.Logger, r Renderer, logger *slog.Logger) *Handler {
	return &Handler{
		session:         session,
		username:        username,
		password:        password,
		recaptchaSite:   recaptchaSiteKey,
		recaptchaSecret: recaptchaSecretKey,
		httpClient:      &http.Client{Timeout: 10 * time.Second},
		attempts:        attempts,
		logger:          logger,
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

	if h.recaptchaSecret != "" && !h.verifyRecaptcha(r.PostForm.Get("g-recaptcha-response"), clientIP(r)) {
		h.logAttempt(r, username, false, "invalid_recaptcha")
		h.render(w, "login", map[string]any{
			"Next":          next,
			"RecaptchaSite": h.recaptchaSite,
			"Error":         "Please complete the reCAPTCHA.",
		})
		return
	}

	if !CheckCredentials(username, password, h.username, h.password) {
		h.logAttempt(r, username, false, "invalid_credentials")
		h.render(w, "login", map[string]any{
			"Next":          next,
			"RecaptchaSite": h.recaptchaSite,
			"Error":         "Invalid username or password.",
		})
		return
	}

	h.logAttempt(r, username, true, "")
	h.session.IssueCookie(w, r)
	http.Redirect(w, r, next, http.StatusSeeOther)
}

// logAttempt records a login attempt for auditing. Failures to write the log
// are reported but never block the login flow.
func (h *Handler) logAttempt(r *http.Request, username string, success bool, reason string) {
	if h.attempts == nil {
		return
	}

	attempt := loginlog.Attempt{
		Username:  username,
		IP:        clientIP(r),
		UserAgent: r.Header.Get("User-Agent"),
		Success:   success,
		Reason:    reason,
		CreatedAt: time.Now().UTC(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.attempts.Log(ctx, attempt); err != nil {
		h.logger.Error("log login attempt", "error", err)
	}
}

// clientIP extracts the request IP from RemoteAddr, stripping the port.
func clientIP(r *http.Request) string {
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}
	return r.RemoteAddr
}

// verifyRecaptcha checks a reCAPTCHA v2 response token against Google's
// siteverify endpoint.
func (h *Handler) verifyRecaptcha(token, ip string) bool {
	if token == "" {
		return false
	}

	form := url.Values{
		"secret":   {h.recaptchaSecret},
		"response": {token},
	}
	if ip != "" {
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
