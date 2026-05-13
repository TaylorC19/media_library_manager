package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	bdom "media_library_manager/internal/domain/barcode"
	"media_library_manager/internal/http/middleware"
	barcodesvc "media_library_manager/internal/service/barcode"
)

// BarcodeHandler serves POST /barcode/lookup (JSON or form).
type BarcodeHandler struct {
	svc *barcodesvc.Service
}

func NewBarcodeHandler(svc *barcodesvc.Service) *BarcodeHandler {
	return &BarcodeHandler{svc: svc}
}

func (h *BarcodeHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := middleware.CurrentUser(r.Context())
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	req, err := parseBarcodeLookupRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	out, err := h.svc.Lookup(r.Context(), user.ID, req)
	if err != nil {
		if err == barcodesvc.ErrInvalidBarcode {
			http.Error(w, "barcode must contain numeric digits or ISBN characters after normalization", http.StatusBadRequest)
			return
		}
		http.Error(w, "lookup failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(out)
}

func parseBarcodeLookupRequest(r *http.Request) (bdom.Request, error) {
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	var out bdom.Request

	if strings.HasPrefix(ct, "application/json") {
		defer r.Body.Close()
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			return out, err
		}
		if len(strings.TrimSpace(string(body))) == 0 {
			return out, errEmptyJSON
		}
		var raw struct {
			Barcode            string `json:"barcode"`
			PreferredMediaType string `json:"preferredMediaType"`
		}
		if err := json.Unmarshal(body, &raw); err != nil {
			return out, err
		}
		out.Barcode = raw.Barcode
		out.PreferredMediaType = raw.PreferredMediaType
		return out, nil
	}

	if err := r.ParseForm(); err != nil {
		return out, err
	}
	out.Barcode = r.FormValue("barcode")
	out.PreferredMediaType = r.FormValue("preferred_media_type")
	if out.PreferredMediaType == "" {
		out.PreferredMediaType = r.FormValue("preferredMediaType")
	}
	return out, nil
}

var errEmptyJSON = errors.New("request body required for JSON")
