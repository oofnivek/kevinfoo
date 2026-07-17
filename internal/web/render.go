// Package web renders HTML templates for the bookmark UI.
package web

import (
	"embed"
	"fmt"
	"html/template"
	"io"
)

//go:embed all:templates
var templatesFS embed.FS

type Renderer struct {
	tmpl *template.Template
}

func NewRenderer() (*Renderer, error) {
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html", "templates/partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}
	return &Renderer{tmpl: tmpl}, nil
}

func (r *Renderer) Render(w io.Writer, name string, data any) error {
	return r.tmpl.ExecuteTemplate(w, name, data)
}
