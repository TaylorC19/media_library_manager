package views

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
)

//go:embed templates/**/*.html
var embeddedTemplates embed.FS

type Renderer struct {
	tmpl *template.Template
}

func NewRenderer(useEmbedded bool) (*Renderer, error) {
	tmplFS, err := templateFS(useEmbedded)
	if err != nil {
		return nil, err
	}

	parsed, err := template.New("all").ParseFS(tmplFS, "templates/**/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	return &Renderer{tmpl: parsed}, nil
}

func (r *Renderer) Render(w io.Writer, page string, data any) error {
	if err := r.tmpl.ExecuteTemplate(w, page, data); err != nil {
		return fmt.Errorf("execute template %s: %w", page, err)
	}
	return nil
}

func templateFS(useEmbedded bool) (fs.FS, error) {
	if useEmbedded {
		return embeddedTemplates, nil
	}

	return fs.Sub(os.DirFS("."), "internal/views")
}
