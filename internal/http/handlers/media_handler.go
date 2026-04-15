package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	domainauth "media_library_manager/internal/domain/auth"
	"media_library_manager/internal/http/middleware"
	mediasvc "media_library_manager/internal/service/media"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MediaHandler struct {
	media *mediasvc.Service
}

func NewMediaHandler(media *mediasvc.Service) *MediaHandler {
	return &MediaHandler{media: media}
}

func (h *MediaHandler) requireUser(w http.ResponseWriter, r *http.Request) *domainauth.User {
	user := middleware.CurrentUser(r.Context())
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil
	}
	return user
}

func (h *MediaHandler) setFlashRedirect(w http.ResponseWriter, r *http.Request, target, level, msgKey string) {
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

func (h *MediaHandler) ImportSubmit(w http.ResponseWriter, r *http.Request) {
	user := h.requireUser(w, r)
	if user == nil {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	locale := strings.TrimSpace(r.FormValue("locale"))
	if locale == "" {
		locale = middleware.LocaleFromContext(r.Context())
	}

	form := mediasvc.ImportForm{
		Bucket:     r.FormValue("bucket"),
		MediaType:  r.FormValue("media_type"),
		Provider:   r.FormValue("provider"),
		ExternalID: r.FormValue("external_id"),
		TMDBKind:   r.FormValue("tmdb_kind"),
	}

	outcome, formErrs, err := h.media.Import(r.Context(), user.ID, form)
	if err != nil {
		switch {
		case errors.Is(err, mediasvc.ErrValidation):
			http.Error(w, strings.Join(formErrs, ", "), http.StatusBadRequest)
		case errors.Is(err, mediasvc.ErrNotFound):
			http.Error(w, "provider record not found", http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if outcome.LibraryEntry != nil {
		target := fmt.Sprintf("/%s/library/%s", locale, outcome.LibraryEntry.ID.Hex())
		flashKey := "search.flash.imported"
		if outcome.WasExistingLibraryEntry {
			flashKey = "search.flash.existingEntry"
		}
		h.setFlashRedirect(w, r, target, "common.flash.info", flashKey)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"mediaRecordId":           outcome.MediaRecord.ID.Hex(),
		"wasExistingMediaRecord":  outcome.WasExistingMediaRecord,
		"wasExistingLibraryEntry": outcome.WasExistingLibraryEntry,
	})
}

func (h *MediaHandler) RefreshSubmit(w http.ResponseWriter, r *http.Request) {
	user := h.requireUser(w, r)
	if user == nil {
		return
	}

	mediaRecordID, ok := parseMediaRecordIDParam(w, r, chi.URLParam(r, "mediaRecordId"))
	if !ok {
		return
	}

	outcome, err := h.media.RefreshScaffold(r.Context(), mediaRecordID)
	if err != nil {
		if errors.Is(err, mediasvc.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(outcome)
}

func parseMediaRecordIDParam(w http.ResponseWriter, r *http.Request, raw string) (primitive.ObjectID, bool) {
	raw = strings.TrimSpace(raw)
	if len(raw) != 24 {
		http.NotFound(w, r)
		return primitive.NilObjectID, false
	}
	id, err := primitive.ObjectIDFromHex(raw)
	if err != nil {
		http.NotFound(w, r)
		return primitive.NilObjectID, false
	}
	return id, true
}
