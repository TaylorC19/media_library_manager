package views

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"strings"
)

//go:embed templates/**/*.html
var embeddedTemplates embed.FS

//go:embed locales/*.json
var embeddedLocales embed.FS

type Renderer struct {
	tmpl         *template.Template
	translations map[string]map[string]any
}

func NewRenderer(useEmbedded bool) (*Renderer, error) {
	tmplFS, err := templateFS(useEmbedded)
	if err != nil {
		return nil, err
	}

	localeMap, err := loadLocales(useEmbedded)
	if err != nil {
		return nil, err
	}

	formatKey := func(v string) string {
		v = strings.TrimSpace(v)
		if v == "" {
			return ""
		}
		return "library.format." + v
	}

	funcs := template.FuncMap{
		"dict": func(pairs ...any) (map[string]any, error) {
			if len(pairs)%2 != 0 {
				return nil, fmt.Errorf("dict expects an even number of arguments")
			}
			out := make(map[string]any, len(pairs)/2)
			for i := 0; i < len(pairs); i += 2 {
				k, ok := pairs[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				out[k] = pairs[i+1]
			}
			return out, nil
		},
		"libraryFormatKey": func(v any) string {
			switch x := v.(type) {
			case *string:
				if x == nil {
					return ""
				}
				return formatKey(*x)
			case string:
				return formatKey(x)
			default:
				return ""
			}
		},
		"t": func(data any, key string) string {
			locale := "en"
			if m, ok := data.(map[string]any); ok {
				if rawLocale, ok := m["Locale"].(string); ok {
					locale = strings.ToLower(strings.TrimSpace(rawLocale))
				}
			}
			if msgs, ok := localeMap[locale]; ok {
				if value, ok := resolveTranslation(msgs, key); ok {
					return value
				}
			}
			if msgs, ok := localeMap["en"]; ok {
				if value, ok := resolveTranslation(msgs, key); ok {
					return value
				}
			}
			return key
		},
	}

	parsed, err := template.New("all").Funcs(funcs).ParseFS(tmplFS, "templates/**/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	return &Renderer{tmpl: parsed, translations: localeMap}, nil
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

func loadLocales(useEmbedded bool) (map[string]map[string]any, error) {
	var localeFS fs.FS
	var err error
	if useEmbedded {
		localeFS, err = fs.Sub(embeddedLocales, "locales")
		if err != nil {
			return nil, fmt.Errorf("embedded locale fs: %w", err)
		}
	} else {
		localeFS, err = fs.Sub(os.DirFS("."), "internal/views/locales")
		if err != nil {
			return nil, fmt.Errorf("locale fs: %w", err)
		}
	}

	entries, err := fs.ReadDir(localeFS, ".")
	if err != nil {
		return nil, fmt.Errorf("read locales: %w", err)
	}

	out := map[string]map[string]any{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		raw, err := fs.ReadFile(localeFS, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read locale %s: %w", entry.Name(), err)
		}
		msgs := map[string]any{}
		if err := json.Unmarshal(raw, &msgs); err != nil {
			return nil, fmt.Errorf("parse locale %s: %w", entry.Name(), err)
		}
		locale := strings.TrimSuffix(entry.Name(), ".json")
		out[locale] = msgs
	}
	if _, ok := out["en"]; !ok {
		return nil, fmt.Errorf("missing required locale file: en.json")
	}
	return out, nil
}

func resolveTranslation(msgs map[string]any, key string) (string, bool) {
	parts := strings.Split(strings.TrimSpace(key), ".")
	if len(parts) == 0 {
		return "", false
	}

	var current any = msgs
	for _, part := range parts {
		obj, ok := current.(map[string]any)
		if !ok {
			return "", false
		}
		value, ok := obj[part]
		if !ok {
			return "", false
		}
		current = value
	}

	text, ok := current.(string)
	if !ok {
		return "", false
	}
	return text, true
}
