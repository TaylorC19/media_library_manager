package handlers

import (
	"net/http"
	"strings"

	domainauth "media_library_manager/internal/domain/auth"
	domainlib "media_library_manager/internal/domain/library"
	domainsearch "media_library_manager/internal/domain/search"
	"media_library_manager/internal/http/middleware"
	searchsvc "media_library_manager/internal/service/search"
	"media_library_manager/internal/views"
)

type SearchHandler struct {
	render *views.Renderer
	search *searchsvc.Service
}

func NewSearchHandler(render *views.Renderer, search *searchsvc.Service) *SearchHandler {
	return &SearchHandler{render: render, search: search}
}

func (h *SearchHandler) baseAppData(r *http.Request, title, pageTitle string) map[string]any {
	user := middleware.CurrentUser(r.Context())
	locale := middleware.LocaleFromContext(r.Context())
	return map[string]any{
		"Title":           title + " - Media Library Manager",
		"PageTitle":       pageTitle,
		"Locale":          locale,
		"User":            user,
		"MediaTypes":      domainlib.MediaTypes,
		"Buckets":         domainlib.Buckets,
		"Formats":         domainlib.Formats,
		"Values":          map[string]string{},
		"FormErrors":      []string{},
		"ContentTemplate": "",
	}
}

func (h *SearchHandler) renderPage(w http.ResponseWriter, page string, data map[string]any) {
	if err := h.render.Render(w, page, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *SearchHandler) consumeFlash(w http.ResponseWriter, r *http.Request, data map[string]any) {
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

func (h *SearchHandler) setFlashRedirect(w http.ResponseWriter, r *http.Request, target, level, msgKey string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    middleware.EncodeFlash(level, msgKey),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   10,
	})
	http.Redirect(w, r, target, http.StatusFound)
}

func (h *SearchHandler) requireUser(w http.ResponseWriter, r *http.Request) *domainauth.User {
	user := middleware.CurrentUser(r.Context())
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil
	}
	return user
}

func (h *SearchHandler) Page(w http.ResponseWriter, r *http.Request) {
	user := h.requireUser(w, r)
	if user == nil {
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	mediaType := strings.TrimSpace(r.URL.Query().Get("media_type"))

	data := h.baseAppData(r, "Search", "")
	data["ContentTemplate"] = "pages/search.content"
	data["Query"] = q
	data["Hits"] = []domainsearch.Hit{}
	data["SearchWarnings"] = []string{}
	data["SearchError"] = ""
	data["Values"] = map[string]string{
		"q":          q,
		"media_type": mediaType,
	}

	if q != "" && mediaType == "" {
		data["SearchError"] = "search.errors.mediaTypeRequired"
	}
	if q == "" {
		h.consumeFlash(w, r, data)
		h.renderPage(w, "pages/search.html", data)
		return
	}
	if mediaType == "" {
		h.consumeFlash(w, r, data)
		h.renderPage(w, "pages/search.html", data)
		return
	}
	if !domainlib.IsMediaType(mediaType) {
		data["SearchError"] = "search.errors.invalidMediaType"
		h.consumeFlash(w, r, data)
		h.renderPage(w, "pages/search.html", data)
		return
	}

	out, err := h.search.Search(r.Context(), mediaType, q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	data["Hits"] = out.Hits
	data["SearchWarnings"] = out.Warnings

	h.consumeFlash(w, r, data)
	h.renderPage(w, "pages/search.html", data)
}
