package handlers

import (
	"net/http"
	"strings"

	domainlib "media_library_manager/internal/domain/library"
	"media_library_manager/internal/http/middleware"
	searchsvc "media_library_manager/internal/service/search"
)

type pendingSearchGroup struct{ MediaType, URL string }

func pendingSearchGroups(locale, query, view string, pageSize int) []pendingSearchGroup {
	groups := make([]pendingSearchGroup, 0, len(domainlib.MediaTypes))
	for _, kind := range domainlib.MediaTypes {
		groups = append(groups, pendingSearchGroup{kind, pageURL("/"+locale+"/search/group", searchPageValues(query, kind, view), 1, pageSize)})
	}
	return groups
}

// Group renders one independent provider lookup, including personal badges.
// The browser can display fast groups without waiting for other providers.
func (h *SearchHandler) Group(w http.ResponseWriter, r *http.Request) {
	if h.requireUser(w, r) == nil {
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	kind := strings.TrimSpace(r.URL.Query().Get("media_type"))
	if query == "" || !domainlib.IsMediaType(kind) {
		http.Error(w, "Invalid search group", http.StatusBadRequest)
		return
	}
	out, err := h.search.Search(r.Context(), kind, query)
	if err != nil {
		http.Error(w, "Could not load search group", http.StatusBadGateway)
		return
	}
	// Provider adapters report transient outages as scoped warnings. Treat an
	// unavailable group as retryable, while preserving key-configuration warnings.
	for _, warning := range out.Warnings {
		if len(out.Hits) == 0 && warning == "search.warnings.providerUnavailable" {
			http.Error(w, "Could not load search group", http.StatusBadGateway)
			return
		}
	}
	locale := middleware.LocaleFromContext(r.Context())
	view := searchView(r)
	groups, _ := buildSearchGroups("/"+locale+"/search", query, parsePageSize(r.URL.Query().Get("page_size")), view, []searchsvc.Group{{MediaType: kind, Outcome: out}})
	if len(groups) == 0 {
		groups = []searchGroup{{MediaType: kind}}
	}
	data := h.baseAppData(r, "search.title")
	data["View"] = view
	data["SearchGroups"] = groups
	data["SearchWarnings"] = []string{}
	h.addSavedStatus(r, data)
	group := data["SearchGroups"].([]searchGroup)[0]
	group.Warnings = append(group.Warnings, data["SearchWarnings"].([]string)...)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.renderPage(w, "partials/search_group", map[string]any{"Page": data, "Group": group})
}
