package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	domainauth "media_library_manager/internal/domain/auth"
	domainlib "media_library_manager/internal/domain/library"
	"media_library_manager/internal/http/middleware"
	libsvc "media_library_manager/internal/service/library"
	"media_library_manager/internal/views"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LibraryHandler struct {
	render  *views.Renderer
	library *libsvc.Service
}

func NewLibraryHandler(render *views.Renderer, library *libsvc.Service) *LibraryHandler {
	return &LibraryHandler{render: render, library: library}
}

func (h *LibraryHandler) baseAppData(r *http.Request, title, pageTitle string) map[string]any {
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

func (h *LibraryHandler) renderPage(w http.ResponseWriter, page string, data map[string]any) {
	if err := h.render.Render(w, page, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *LibraryHandler) consumeFlash(w http.ResponseWriter, r *http.Request, data map[string]any) {
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

func (h *LibraryHandler) setFlashRedirect(w http.ResponseWriter, r *http.Request, target, level, msgKey string) {
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

func (h *LibraryHandler) requireUser(w http.ResponseWriter, r *http.Request) *domainauth.User {
	user := middleware.CurrentUser(r.Context())
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil
	}
	return user
}

func (h *LibraryHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	user := h.requireUser(w, r)
	if user == nil {
		return
	}

	catalogCount, wishlistCount, err := h.library.DashboardStats(r.Context(), user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := h.baseAppData(r, "Dashboard", "")
	data["ContentTemplate"] = "pages/dashboard.content"
	data["CatalogCount"] = catalogCount
	data["WishlistCount"] = wishlistCount
	h.consumeFlash(w, r, data)
	h.renderPage(w, "pages/dashboard.html", data)
}

func (h *LibraryHandler) Catalog(w http.ResponseWriter, r *http.Request) {
	h.listBucket(w, r, domainlib.BucketCatalog, "Catalog", "pages/catalog.content")
}

func (h *LibraryHandler) Wishlist(w http.ResponseWriter, r *http.Request) {
	h.listBucket(w, r, domainlib.BucketWishlist, "Wishlist", "pages/wishlist.content")
}

func (h *LibraryHandler) listBucket(w http.ResponseWriter, r *http.Request, bucket, title, contentTpl string) {
	user := h.requireUser(w, r)
	if user == nil {
		return
	}
	locale := middleware.LocaleFromContext(r.Context())
	page := parsePage(r.URL.Query().Get("page"))
	pageSize := parsePageSize(r.URL.Query().Get("page_size"))
	filters := libsvc.ListBucketFilters{
		MediaType: strings.TrimSpace(r.URL.Query().Get("media_type")),
		Format:    strings.TrimSpace(r.URL.Query().Get("format")),
		Query:     strings.TrimSpace(r.URL.Query().Get("q")),
	}
	result, err := h.library.ListBucket(r.Context(), user.ID, bucket, filters, page, pageSize)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	listPath := "/" + locale + "/" + bucket
	data := h.baseAppData(r, title, "")
	data["ContentTemplate"] = contentTpl
	data["List"] = result.Items
	data["ListTotal"] = result.TotalItems
	data["Bucket"] = bucket
	data["ListPath"] = listPath
	data["Pagination"] = buildPagination(listPath, bucketValues(filters), page, pageSize, result.TotalItems)
	data["Values"] = map[string]string{
		"q":          filters.Query,
		"media_type": filters.MediaType,
		"format":     filters.Format,
		"page_size":  strconv.Itoa(pageSize),
	}
	h.renderBucketResponse(w, r, data)
}

func (h *LibraryHandler) Detail(w http.ResponseWriter, r *http.Request) {
	user := h.requireUser(w, r)
	if user == nil {
		return
	}
	entryID, ok := parseObjectIDParam(w, r, chi.URLParam(r, "entryId"))
	if !ok {
		return
	}
	detail, err := h.library.GetEntryDetail(r.Context(), user.ID, entryID)
	if err != nil {
		if errors.Is(err, libsvc.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := h.baseAppData(r, detail.Media.Title, "")
	data["ContentTemplate"] = "pages/library_detail.content"
	data["Detail"] = detail
	data["CanEditMedia"] = detail.Media.Source == domainlib.SourceManual
	h.consumeFlash(w, r, data)
	h.renderPage(w, "pages/library_detail.html", data)
}

func (h *LibraryHandler) NewForm(w http.ResponseWriter, r *http.Request) {
	user := h.requireUser(w, r)
	if user == nil {
		return
	}
	data := h.baseAppData(r, "Add item", "")
	data["ContentTemplate"] = "pages/library_form.content"
	data["FormAction"] = "/library"
	data["IsEdit"] = false
	bucket := strings.TrimSpace(r.URL.Query().Get("bucket"))
	if bucket == "" {
		bucket = domainlib.BucketCatalog
	}
	if !domainlib.IsBucket(bucket) {
		bucket = domainlib.BucketCatalog
	}
	data["Values"] = map[string]string{"bucket": bucket}
	data["CanEditMedia"] = true
	h.consumeFlash(w, r, data)
	h.renderPage(w, "pages/library_form.html", data)
}

func (h *LibraryHandler) EditForm(w http.ResponseWriter, r *http.Request) {
	user := h.requireUser(w, r)
	if user == nil {
		return
	}
	entryID, ok := parseObjectIDParam(w, r, chi.URLParam(r, "entryId"))
	if !ok {
		return
	}
	detail, err := h.library.GetEntryDetail(r.Context(), user.ID, entryID)
	if err != nil {
		if errors.Is(err, libsvc.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	values := map[string]string{
		"bucket":        detail.Entry.Bucket,
		"media_type":    detail.Entry.MediaType,
		"title":         detail.Media.Title,
		"format":        formatString(detail.Entry.Format),
		"barcode":       stringVal(detail.Entry.Barcode),
		"purchase_date": stringVal(detail.Entry.PurchaseDate),
		"notes":         stringVal(detail.Entry.Notes),
		"tags":          strings.Join(detail.Entry.Tags, ", "),
	}
	if detail.Media.Year != nil {
		values["year"] = strconv.Itoa(int(*detail.Media.Year))
	}

	data := h.baseAppData(r, "Edit item", "")
	data["ContentTemplate"] = "pages/library_form.content"
	data["FormAction"] = "/library/" + detail.Entry.ID.Hex() + "/update"
	data["IsEdit"] = true
	data["EntryID"] = detail.Entry.ID.Hex()
	data["Values"] = values
	data["CanEditMedia"] = detail.Media.Source == domainlib.SourceManual
	h.consumeFlash(w, r, data)
	h.renderPage(w, "pages/library_form.html", data)
}

func (h *LibraryHandler) CreateSubmit(w http.ResponseWriter, r *http.Request) {
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

	form := libsvc.ManualCreateForm{
		Bucket:       r.FormValue("bucket"),
		MediaType:    r.FormValue("media_type"),
		Title:        r.FormValue("title"),
		Year:         parseYearField(r.FormValue("year")),
		Format:       r.FormValue("format"),
		Barcode:      r.FormValue("barcode"),
		PurchaseDate: r.FormValue("purchase_date"),
		Notes:        r.FormValue("notes"),
		Tags:         r.FormValue("tags"),
	}

	entryID, formErrs, err := h.library.CreateManual(r.Context(), user.ID, form)
	if err != nil {
		if errors.Is(err, libsvc.ErrValidation) {
			h.renderFormErrors(w, r, "/library", false, "", formValuesFromCreate(form), formErrs)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	target := fmt.Sprintf("/%s/library/%s", locale, entryID.Hex())
	h.setFlashRedirect(w, r, target, "common.flash.info", "library.flash.created")
}

func (h *LibraryHandler) AttachSubmit(w http.ResponseWriter, r *http.Request) {
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
	raw := strings.TrimSpace(r.FormValue("media_record_id"))
	if len(raw) != 24 {
		h.setFlashRedirect(w, r, "/"+locale+"/scan", "common.flash.error", "library.errors.invalidMediaRecordId")
		return
	}
	mediaRecordID, err := primitive.ObjectIDFromHex(raw)
	if err != nil {
		h.setFlashRedirect(w, r, "/"+locale+"/scan", "common.flash.error", "library.errors.invalidMediaRecordId")
		return
	}
	bucket := strings.TrimSpace(r.FormValue("bucket"))
	barcode := strings.TrimSpace(r.FormValue("barcode"))

	entryID, wasExisting, formErrs, err := h.library.AttachFromMediaRecord(r.Context(), user.ID, mediaRecordID, bucket, barcode)
	if err != nil {
		if errors.Is(err, libsvc.ErrValidation) {
			msgKey := "common.flash.serverError"
			if len(formErrs) > 0 {
				msgKey = formErrs[0]
			}
			h.setFlashRedirect(w, r, "/"+locale+"/scan", "common.flash.error", msgKey)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	flashKey := "library.flash.addedToLibrary"
	if wasExisting {
		flashKey = "search.flash.existingEntry"
	}
	target := fmt.Sprintf("/%s/library/%s", locale, entryID.Hex())
	h.setFlashRedirect(w, r, target, "common.flash.info", flashKey)
}

func (h *LibraryHandler) UpdateSubmit(w http.ResponseWriter, r *http.Request) {
	user := h.requireUser(w, r)
	if user == nil {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	locale := strings.TrimSpace(r.FormValue("locale"))
	if locale == "" {
		locale = middleware.LocaleFromContext(r.Context())
	}
	entryID, ok := parseObjectIDParam(w, r, chi.URLParam(r, "entryId"))
	if !ok {
		return
	}

	form := libsvc.ManualUpdateForm{
		MediaType:    r.FormValue("media_type"),
		Title:        r.FormValue("title"),
		Year:         parseYearField(r.FormValue("year")),
		Bucket:       r.FormValue("bucket"),
		Format:       r.FormValue("format"),
		Barcode:      r.FormValue("barcode"),
		PurchaseDate: r.FormValue("purchase_date"),
		Notes:        r.FormValue("notes"),
		Tags:         r.FormValue("tags"),
	}

	formErrs, err := h.library.UpdateEntry(r.Context(), user.ID, entryID, form)
	if err != nil {
		if errors.Is(err, libsvc.ErrValidation) {
			action := "/library/" + entryID.Hex() + "/update"
			h.renderFormErrors(w, r, action, true, entryID.Hex(), formValuesFromUpdate(form), formErrs)
			return
		}
		if errors.Is(err, libsvc.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	target := fmt.Sprintf("/%s/library/%s", locale, entryID.Hex())
	h.setFlashRedirect(w, r, target, "common.flash.info", "library.flash.updated")
}

func (h *LibraryHandler) DeleteSubmit(w http.ResponseWriter, r *http.Request) {
	user := h.requireUser(w, r)
	if user == nil {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	locale := strings.TrimSpace(r.FormValue("locale"))
	if locale == "" {
		locale = middleware.LocaleFromContext(r.Context())
	}
	entryID, ok := parseObjectIDParam(w, r, chi.URLParam(r, "entryId"))
	if !ok {
		return
	}
	if err := h.library.DeleteEntry(r.Context(), user.ID, entryID); err != nil {
		if errors.Is(err, libsvc.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	target := fmt.Sprintf("/%s/catalog", locale)
	h.setFlashRedirect(w, r, target, "common.flash.info", "library.flash.deleted")
}

func (h *LibraryHandler) renderFormErrors(w http.ResponseWriter, r *http.Request, formAction string, isEdit bool, entryID string, values map[string]string, formErrs []string) {
	data := h.baseAppData(r, "Add item", "")
	if isEdit {
		data["Title"] = "Edit item - Media Library Manager"
		data["PageTitle"] = "Edit item"
	}
	data["ContentTemplate"] = "pages/library_form.content"
	data["FormAction"] = formAction
	data["IsEdit"] = isEdit
	data["EntryID"] = entryID
	data["Values"] = values
	data["FormErrors"] = formErrs
	canEditMedia := true
	if isEdit && entryID != "" {
		user := middleware.CurrentUser(r.Context())
		if user != nil {
			if oid, err := primitive.ObjectIDFromHex(entryID); err == nil {
				if d, err := h.library.GetEntryDetail(r.Context(), user.ID, oid); err == nil && d != nil {
					canEditMedia = d.Media.Source == domainlib.SourceManual
				}
			}
		}
	}
	data["CanEditMedia"] = canEditMedia
	h.renderPage(w, "pages/library_form.html", data)
}

func (h *LibraryHandler) renderBucketResponse(w http.ResponseWriter, r *http.Request, data map[string]any) {
	if isHTMX(r) {
		h.renderPage(w, "partials/library_list_section", data)
		return
	}
	h.consumeFlash(w, r, data)
	h.renderPage(w, "pages/library_list.html", data)
}

func bucketValues(filters libsvc.ListBucketFilters) url.Values {
	values := url.Values{}
	if filters.Query != "" {
		values.Set("q", filters.Query)
	}
	if filters.MediaType != "" {
		values.Set("media_type", filters.MediaType)
	}
	if filters.Format != "" {
		values.Set("format", filters.Format)
	}
	return values
}

func formValuesFromCreate(f libsvc.ManualCreateForm) map[string]string {
	return map[string]string{
		"bucket":        f.Bucket,
		"media_type":    f.MediaType,
		"title":         f.Title,
		"year":          yearString(f.Year),
		"format":        f.Format,
		"barcode":       f.Barcode,
		"purchase_date": f.PurchaseDate,
		"notes":         f.Notes,
		"tags":          f.Tags,
	}
}

func formValuesFromUpdate(f libsvc.ManualUpdateForm) map[string]string {
	return map[string]string{
		"bucket":        f.Bucket,
		"media_type":    f.MediaType,
		"title":         f.Title,
		"year":          yearString(f.Year),
		"format":        f.Format,
		"barcode":       f.Barcode,
		"purchase_date": f.PurchaseDate,
		"notes":         f.Notes,
		"tags":          f.Tags,
	}
}

func yearString(y *int) string {
	if y == nil {
		return ""
	}
	return strconv.Itoa(*y)
}

func parseYearField(s string) *int {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	y, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &y
}

func parseObjectIDParam(w http.ResponseWriter, r *http.Request, raw string) (primitive.ObjectID, bool) {
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

func formatString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func stringVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
