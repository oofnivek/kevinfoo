// Package web renders HTML templates for the bookmark UI.
package web

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"strings"
)

//go:embed all:templates
var templatesFS embed.FS

var funcMap = template.FuncMap{
	// initial returns the uppercased first character of s, for row avatars.
	"initial": func(s string) string {
		for _, r := range s {
			return strings.ToUpper(string(r))
		}
		return ""
	},
}

type Renderer struct {
	tmpl *template.Template
}

func NewRenderer() (*Renderer, error) {
	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templatesFS, "templates/*.html", "templates/partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}
	return &Renderer{tmpl: tmpl}, nil
}

func (r *Renderer) Render(w io.Writer, name string, data any) error {
	return r.tmpl.ExecuteTemplate(w, name, data)
}
