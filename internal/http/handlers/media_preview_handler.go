package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	domainlib "media_library_manager/internal/domain/library"
	"media_library_manager/internal/http/middleware"
	mediasvc "media_library_manager/internal/service/media"
)

// Preview is read-only. Saving requires a separate POST from the preview actions.
func (h *MediaHandler) Preview(w http.ResponseWriter, r *http.Request) {
	if h.requireUser(w, r) == nil {
		return
	}
	query := r.URL.Query()
	form := mediasvc.ImportForm{
		Provider: strings.TrimSpace(query.Get("provider")), ExternalID: strings.TrimSpace(query.Get("external_id")),
		MediaType: strings.TrimSpace(query.Get("media_type")), TMDBKind: strings.TrimSpace(query.Get("tmdb_kind")),
	}
	if form.Provider == "openlibrary" {
		form.Provider = "open_library"
	}
	var record *domainlib.MediaRecord
	var err error
	localID := query.Get("media_record_id")
	if localID != "" {
		id, parseErr := primitive.ObjectIDFromHex(localID)
		if parseErr != nil {
			http.NotFound(w, r)
			return
		}
		record, err = h.media.PreviewLocal(r.Context(), id)
	} else {
		record, err = h.media.Preview(r.Context(), form)
	}
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, mediasvc.ErrValidation) {
			status = http.StatusBadRequest
		}
		if errors.Is(err, mediasvc.ErrNotFound) {
			status = http.StatusNotFound
		}
		http.Error(w, "Could not load item details", status)
		return
	}
	saved, err := h.media.SavedEntries(r.Context(), middleware.CurrentUser(r.Context()).ID, record, form)
	if err != nil {
		http.Error(w, "Could not load saved-item status", http.StatusInternalServerError)
		return
	}
	locale := middleware.LocaleFromContext(r.Context())
	data := map[string]any{
		"Locale": locale, "Path": r.URL.Path, "User": middleware.CurrentUser(r.Context()),
		"TitleKey": "search.dialog.title", "ContentTemplate": "pages/search_detail.content",
		"Record": record, "Fields": previewFields(record), "Import": form, "LocalID": localID,
		"Barcode": query.Get("barcode"), "Buckets": domainlib.Buckets, "SavedByBucket": saved, "Formats": domainlib.FormatsForMediaType(record.MediaType),
	}
	page := "pages/search_detail.html"
	if query.Get("fragment") == "1" {
		page = "pages/search_detail.content"
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.render.Render(w, page, data); err != nil {
		http.Error(w, "Could not render item details", http.StatusInternalServerError)
	}
}

type previewField struct{ Key, Value string }

func previewFields(record *domainlib.MediaRecord) []previewField {
	var fields []previewField
	if record == nil {
		return fields
	}
	if record.ReleaseDate != nil && *record.ReleaseDate != "" {
		fields = append(fields, previewField{"releaseDate", *record.ReleaseDate})
	}
	for _, key := range []string{"authors", "directors", "creators", "artists", "developers", "publishers", "genres", "platforms", "cast", "runtimeMinutes", "seasons", "episodes", "label", "catalogNumber", "releaseCountry", "trackCount"} {
		var value string
		switch v := record.Details[key].(type) {
		case string:
			value = v
		case *string:
			if v != nil {
				value = *v
			}
		case *int32:
			if v != nil {
				value = fmt.Sprint(*v)
			}
		case int, int32, int64, float64:
			value = fmt.Sprint(v)
		case []string:
			value = strings.Join(v, ", ")
		case bson.A:
			var parts []string
			for _, part := range v {
				if text, ok := part.(string); ok {
					parts = append(parts, text)
				}
			}
			value = strings.Join(parts, ", ")
		}
		if strings.TrimSpace(value) != "" {
			fields = append(fields, previewField{key, value})
		}
	}
	return fields
}
