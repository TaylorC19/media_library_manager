package handlers

import (
	"net/http"
	"net/url"
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

func (h *SearchHandler) baseAppData(r *http.Request, titleKey string) map[string]any {
	user := middleware.CurrentUser(r.Context())
	locale := middleware.LocaleFromContext(r.Context())
	return map[string]any{
		"TitleKey":        titleKey,
		"Locale":          locale,
		"Path":            r.URL.Path,
		"LocalePath":      r.URL.RequestURI(),
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
	page := parsePage(r.URL.Query().Get("page"))
	pageSize := parsePageSize(r.URL.Query().Get("page_size"))
	locale := middleware.LocaleFromContext(r.Context())
	basePath := "/" + locale + "/search"

	data := h.baseAppData(r, "search.title")
	data["ContentTemplate"] = "pages/search.content"
	view := searchView(r)
	data["View"] = view
	data["ViewURLs"] = map[string]string{
		"list": libraryViewURL(r, basePath, "list"),
		"grid": libraryViewURL(r, basePath, "grid"),
	}
	scanOpen := r.URL.Query().Get("mode") == "barcode"
	data["ScanOpen"] = scanOpen
	data["PageScripts"] = []string{"/static/js/vendor/zxing-browser.min.js", "/static/js/scan.js", "/static/js/search.js"}
	data["Query"] = q
	data["Hits"] = []domainsearch.Hit{}
	data["AllMedia"] = mediaType == ""
	data["SearchGroups"] = []searchGroup{}
	data["HasResults"] = false
	data["SearchWarnings"] = []string{}
	data["SearchError"] = ""
	data["Values"] = map[string]string{
		"q":          q,
		"media_type": mediaType,
		"page_size":  r.URL.Query().Get("page_size"),
	}
	data["Pagination"] = buildPagination(basePath, searchPageValues(q, mediaType, view), page, pageSize, 0)
	data["SearchPath"] = basePath

	if q == "" || scanOpen {
		h.renderSearchResponse(w, r, data)
		return
	}
	if mediaType != "" && !domainlib.IsMediaType(mediaType) {
		data["SearchError"] = "search.errors.invalidMediaType"
		h.renderSearchResponse(w, r, data)
		return
	}
	if mediaType == "" {
		groups, err := h.search.SearchAll(r.Context(), q)
		if err != nil {
			http.Error(w, "Could not complete search", http.StatusBadGateway)
			return
		}
		previews, hasResults := buildSearchGroups(basePath, q, pageSize, view, groups)
		data["SearchGroups"] = previews
		data["HasResults"] = hasResults
		h.renderSearchResponse(w, r, data)
		return
	}

	out, err := h.search.Search(r.Context(), mediaType, q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	totalItems := len(out.Hits)
	start, end := pageBounds(totalItems, page, pageSize)
	data["Hits"] = out.Hits[start:end]
	data["SearchWarnings"] = out.Warnings
	data["Pagination"] = buildPagination(basePath, searchPageValues(q, mediaType, view), page, pageSize, totalItems)

	h.renderSearchResponse(w, r, data)
}

func (h *SearchHandler) renderSearchResponse(w http.ResponseWriter, r *http.Request, data map[string]any) {
	if isHTMX(r) {
		h.renderPage(w, "partials/search_results", data)
		return
	}
	h.consumeFlash(w, r, data)
	h.renderPage(w, "pages/search.html", data)
}

func searchValues(query, mediaType string) url.Values {
	values := url.Values{}
	if query != "" {
		values.Set("q", query)
	}
	if mediaType != "" {
		values.Set("media_type", mediaType)
	}
	return values
}

const searchGroupPreviewSize = 3

type searchGroup struct {
	MediaType string
	Hits      []domainsearch.Hit
	Warnings  []string
	Total     int
	SeeAllURL string
}

func buildSearchGroups(path, query string, pageSize int, view string, groups []searchsvc.Group) ([]searchGroup, bool) {
	previews := make([]searchGroup, 0, len(groups))
	hasResults := false
	for _, group := range groups {
		if len(group.Hits) == 0 && len(group.Warnings) == 0 {
			continue
		}
		count := len(group.Hits)
		if count > searchGroupPreviewSize {
			count = searchGroupPreviewSize
		}
		if count > 0 {
			hasResults = true
		}
		previews = append(previews, searchGroup{
			MediaType: group.MediaType, Hits: group.Hits[:count], Warnings: group.Warnings,
			Total: len(group.Hits), SeeAllURL: pageURL(path, searchPageValues(query, group.MediaType, view), 1, pageSize),
		})
	}
	return previews, hasResults
}

func searchView(r *http.Request) string {
	if r.URL.Query().Get("view") == "list" {
		return "list"
	}
	return "grid"
}

func searchPageValues(query, mediaType, view string) url.Values {
	values := searchValues(query, mediaType)
	values.Set("view", view)
	return values
}
