package handlers

import (
	"net/http"

	domainauth "media_library_manager/internal/domain/auth"
	domainlib "media_library_manager/internal/domain/library"
	"media_library_manager/internal/http/middleware"
	"media_library_manager/internal/views"
)

type SettingsHandler struct {
	render *views.Renderer
}

func NewSettingsHandler(render *views.Renderer) *SettingsHandler {
	return &SettingsHandler{render: render}
}

func (h *SettingsHandler) baseAppData(r *http.Request, titleKey string) map[string]any {
	user := middleware.CurrentUser(r.Context())
	locale := middleware.LocaleFromContext(r.Context())
	return map[string]any{
		"TitleKey":        titleKey,
		"Locale":          locale,
		"Path":            r.URL.Path,
		"User":            user,
		"MediaTypes":      domainlib.MediaTypes,
		"Buckets":         domainlib.Buckets,
		"Formats":         domainlib.Formats,
		"Values":          map[string]string{},
		"FormErrors":      []string{},
		"ContentTemplate": "",
	}
}

func (h *SettingsHandler) renderPage(w http.ResponseWriter, page string, data map[string]any) {
	if err := h.render.Render(w, page, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *SettingsHandler) consumeFlash(w http.ResponseWriter, r *http.Request, data map[string]any) {
	c, err := r.Cookie("flash")
	if err != nil {
		return
	}
	flash := middleware.DecodeFlash(c.Value)
	if flash != nil {
		data["Flash"] = flash
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *SettingsHandler) requireUser(w http.ResponseWriter, r *http.Request) *domainauth.User {
	user := middleware.CurrentUser(r.Context())
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil
	}
	return user
}

func (h *SettingsHandler) Page(w http.ResponseWriter, r *http.Request) {
	user := h.requireUser(w, r)
	if user == nil {
		return
	}

	data := h.baseAppData(r, "settings.title")
	data["ContentTemplate"] = "pages/settings.content"
	data["CurrentLanguageKey"] = "common.languageSwitcher." + data["Locale"].(string)
	h.consumeFlash(w, r, data)
	h.renderPage(w, "pages/settings.html", data)
}
