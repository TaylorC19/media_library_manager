package handlers

import (
	"media_library_manager/internal/http/middleware"
	"net/http"
)

// LegacyScanPage preserves bookmarked scan links within the Search workspace.
// Authentication is enforced by the protected route group.
func LegacyScanPage(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	values.Set("mode", "barcode")
	locale := middleware.LocaleFromContext(r.Context())
	http.Redirect(w, r, "/"+locale+"/search?"+values.Encode(), http.StatusFound)
}
