package push

import (
	"embed"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
)

//go:embed sw.js
var serviceWorker []byte

// manifest makes the page installable as a web app, which iOS requires
// before it allows push: the site must be added to the Home Screen and
// opened from there.
//
//go:embed manifest.webmanifest
var manifest []byte

//go:embed static
var staticFiles embed.FS

const maxRequestBody = 8 << 10

// Handler serves the push notifications page and its JSON endpoints.
type Handler struct {
	svc    *Service
	render func(w io.Writer, name string, data any)
	logger *slog.Logger
}

// Renderer renders a named template to w.
type Renderer interface {
	Render(w io.Writer, name string, data any) error
}

func NewHandler(svc *Service, r Renderer, logger *slog.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger,
		render: func(w io.Writer, name string, data any) {
			if err := r.Render(w, name, data); err != nil {
				logger.Error("render template", "template", name, "error", err)
			}
		},
	}
}

func (h *Handler) Page(w http.ResponseWriter, r *http.Request) {
	h.render(w, "push-notifications", map[string]any{
		"Enabled":   h.svc.Enabled(),
		"PublicKey": h.svc.PublicKey(),
	})
}

func (h *Handler) ServiceWorker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(serviceWorker)
}

func (h *Handler) Manifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/manifest+json")
	w.Write(manifest)
}

// Static serves the app icons under /push/static/.
func (h *Handler) Static() http.Handler {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err) // the embedded directory is fixed at build time
	}
	return http.StripPrefix("/push/static/", http.FileServerFS(sub))
}

// subscriptionJSON matches the shape of PushSubscription.toJSON().
type subscriptionJSON struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

type topicJSON struct {
	UserID string `json:"userId"`
	Topic  string `json:"topic"`
}

func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	var body struct {
		topicJSON
		Subscription subscriptionJSON `json:"subscription"`
	}
	if !h.decode(w, r, &body) {
		return
	}

	err := h.svc.Subscribe(r.Context(), Subscription{
		Endpoint: body.Subscription.Endpoint,
		UserID:   body.UserID,
		Topic:    body.Topic,
		P256dh:   body.Subscription.Keys.P256dh,
		Auth:     body.Subscription.Keys.Auth,
	})
	if err != nil {
		h.error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	var body struct {
		topicJSON
		Endpoint string `json:"endpoint"`
	}
	if !h.decode(w, r, &body) {
		return
	}

	if err := h.svc.Unsubscribe(r.Context(), body.Endpoint, body.UserID, body.Topic); err != nil {
		h.error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Topics lists the user ID + topic pairs a browser is registered under. It is
// a POST so the endpoint, which is enough to push to the browser's push
// service, stays out of URLs and access logs.
func (h *Handler) Topics(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Endpoint string `json:"endpoint"`
	}
	if !h.decode(w, r, &body) {
		return
	}

	subs, err := h.svc.Topics(r.Context(), body.Endpoint)
	if err != nil {
		h.error(w, err)
		return
	}

	topics := make([]topicJSON, 0, len(subs))
	for _, s := range subs {
		topics = append(topics, topicJSON{UserID: s.UserID, Topic: s.Topic})
	}
	h.writeJSON(w, topics)
}

// Send is the public send API. It accepts a JSON body
// ({"userId", "topic", "title", "message"}) or the same fields form-encoded,
// so it can be called from scripts with a plain `curl -d`.
func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	var body struct {
		topicJSON
		Title   string `json:"title"`
		Message string `json:"message"`
	}
	if mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mediaType == "application/x-www-form-urlencoded" {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form body", http.StatusBadRequest)
			return
		}
		body.UserID = r.PostForm.Get("userId")
		body.Topic = r.PostForm.Get("topic")
		body.Title = r.PostForm.Get("title")
		body.Message = r.PostForm.Get("message")
	} else if !h.decode(w, r, &body) {
		return
	}

	result, err := h.svc.Send(r.Context(), body.UserID, body.Topic, Message{Title: body.Title, Body: body.Message})
	if err != nil {
		h.error(w, err)
		return
	}
	h.logger.Info("sent push notification", "sent", result.Sent, "failed", result.Failed, "removed", result.Removed)

	h.writeJSON(w, result)
}

func (h *Handler) decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBody)).Decode(v); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return false
	}
	return true
}

func (h *Handler) writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) error(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, ErrNoSubscribers):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, ErrDisabled):
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	default:
		h.serverError(w, err)
	}
}

func (h *Handler) serverError(w http.ResponseWriter, err error) {
	h.logger.Error("push handler", "error", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
