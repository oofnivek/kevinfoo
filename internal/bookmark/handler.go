package bookmark

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
)

// Handler wires HTTP requests to the bookmark service and renders HTML
// fragments for htmx.
type Handler struct {
	svc    Service
	render func(w http.ResponseWriter, name string, data any)
	logger *slog.Logger
}

// Renderer renders a named template to w.
type Renderer interface {
	Render(w io.Writer, name string, data any) error
}

func NewHandler(svc Service, r Renderer, logger *slog.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger,
		render: func(w http.ResponseWriter, name string, data any) {
			if err := r.Render(w, name, data); err != nil {
				logger.Error("render template", "template", name, "error", err)
			}
		},
	}
}

type formData struct {
	Bookmark    Bookmark
	Method      string
	Action      string
	Target      string
	Swap        string
	CancelURL   string
	FieldSuffix string
	Error       string
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	bookmarks, err := h.svc.List(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		h.serverError(w, err)
		return
	}
	h.render(w, "index", map[string]any{"Bookmarks": bookmarks})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	bookmarks, err := h.svc.List(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		h.serverError(w, err)
		return
	}
	h.render(w, "bookmark-list", map[string]any{"Bookmarks": bookmarks})
}

func (h *Handler) NewForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, "bookmark-form", formData{
		Method:      "post",
		Action:      "/bookmarks",
		Target:      "#bookmark-form-slot",
		Swap:        "innerHTML",
		CancelURL:   "/bookmarks/cancel",
		FieldSuffix: "new",
	})
}

func (h *Handler) CancelForm(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.badRequest(w, err)
		return
	}

	_, err := h.svc.Create(r.Context(), r.PostForm.Get("title"), r.PostForm.Get("url"), r.PostForm.Get("description"), r.PostForm.Get("tags"))
	if errors.Is(err, ErrValidation) {
		h.render(w, "bookmark-form", formData{
			Bookmark:    formBookmark(r),
			Method:      "post",
			Action:      "/bookmarks",
			Target:      "#bookmark-form-slot",
			Swap:        "innerHTML",
			CancelURL:   "/bookmarks/cancel",
			FieldSuffix: "new",
			Error:       err.Error(),
		})
		return
	}
	if err != nil {
		h.serverError(w, err)
		return
	}

	bookmarks, err := h.svc.List(r.Context(), "")
	if err != nil {
		h.serverError(w, err)
		return
	}

	fmt.Fprint(w, `<div id="bookmark-form-slot" hx-swap-oob="true"></div>`)
	h.render(w, "bookmark-list", map[string]any{"Bookmarks": bookmarks, "OOB": true})
}

func (h *Handler) EditForm(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r)
	if err != nil {
		h.badRequest(w, err)
		return
	}

	b, err := h.svc.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		http.Error(w, "bookmark not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.serverError(w, err)
		return
	}

	h.render(w, "bookmark-form", formData{
		Bookmark:    b,
		Method:      "put",
		Action:      fmt.Sprintf("/bookmarks/%d", b.ID),
		Target:      fmt.Sprintf("#bookmark-form-%d", b.ID),
		Swap:        "outerHTML",
		CancelURL:   fmt.Sprintf("/bookmarks/%d", b.ID),
		FieldSuffix: strconv.FormatInt(b.ID, 10),
	})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r)
	if err != nil {
		h.badRequest(w, err)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.badRequest(w, err)
		return
	}

	b, err := h.svc.Update(r.Context(), id, r.PostForm.Get("title"), r.PostForm.Get("url"), r.PostForm.Get("description"), r.PostForm.Get("tags"))
	if errors.Is(err, ErrValidation) {
		formB := formBookmark(r)
		formB.ID = id
		h.render(w, "bookmark-form", formData{
			Bookmark:    formB,
			Method:      "put",
			Action:      fmt.Sprintf("/bookmarks/%d", id),
			Target:      fmt.Sprintf("#bookmark-form-%d", id),
			Swap:        "outerHTML",
			CancelURL:   fmt.Sprintf("/bookmarks/%d", id),
			FieldSuffix: strconv.FormatInt(id, 10),
			Error:       err.Error(),
		})
		return
	}
	if errors.Is(err, ErrNotFound) {
		http.Error(w, "bookmark not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.serverError(w, err)
		return
	}

	h.render(w, "bookmark-row", b)
}

func (h *Handler) Row(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r)
	if err != nil {
		h.badRequest(w, err)
		return
	}

	b, err := h.svc.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		http.Error(w, "bookmark not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.serverError(w, err)
		return
	}

	h.render(w, "bookmark-row", b)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r)
	if err != nil {
		h.badRequest(w, err)
		return
	}

	err = h.svc.Delete(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		http.Error(w, "bookmark not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.serverError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func formBookmark(r *http.Request) Bookmark {
	return Bookmark{
		Title:       r.PostForm.Get("title"),
		URL:         r.PostForm.Get("url"),
		Description: r.PostForm.Get("description"),
		Tags:        r.PostForm.Get("tags"),
	}
}

func idFromPath(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func (h *Handler) badRequest(w http.ResponseWriter, err error) {
	h.logger.Warn("bad request", "error", err)
	http.Error(w, "bad request", http.StatusBadRequest)
}

func (h *Handler) serverError(w http.ResponseWriter, err error) {
	h.logger.Error("internal error", "error", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
