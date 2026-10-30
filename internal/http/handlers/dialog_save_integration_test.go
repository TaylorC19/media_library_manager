package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"media_library_manager/internal/config"
	domainauth "media_library_manager/internal/domain/auth"
	domainlib "media_library_manager/internal/domain/library"
	"media_library_manager/internal/http/middleware"
	"media_library_manager/internal/repository"
	libsvc "media_library_manager/internal/service/library"
	mediasvc "media_library_manager/internal/service/media"
	"media_library_manager/internal/service/reliability"
	"media_library_manager/internal/views"
)

func TestDialogSaveResponsesStayPersonalAndIdempotent(t *testing.T) {
	database := regressionDatabase(t)
	ctx := context.Background()
	owner, other := primitive.NewObjectID(), primitive.NewObjectID()
	media := repository.NewMediaRecordsRepository(database)
	entries := repository.NewLibraryEntriesRepository(database)
	if err := entries.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	record := &domainlib.MediaRecord{Title: "Test dialog book", MediaType: "book", Source: "provider", ProviderRefs: bson.M{"openLibrary": bson.M{"workKey": "OL1W"}}}
	if err := media.Insert(ctx, record); err != nil {
		t.Fatal(err)
	}
	otherCopy := &domainlib.LibraryEntry{UserID: other, MediaRecordID: record.ID, MediaType: "book", Bucket: "catalog"}
	if err := entries.Insert(ctx, otherCopy); err != nil {
		t.Fatal(err)
	}
	render, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{}
	mh := NewMediaHandler(render, mediasvc.NewService(cfg, media, entries, nil, reliability.NewService(cfg), nil))
	lh := NewLibraryHandler(render, libsvc.NewService(media, entries))
	request := func(path, body, role string) *http.Request {
		r := httptest.NewRequest("POST", path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("X-Search-Dialog", "1")
		return r.WithContext(middleware.WithUser(r.Context(), &domainauth.User{ID: owner, Role: role}))
	}
	for _, method := range []http.HandlerFunc{mh.ImportSubmit, lh.AttachSubmit} {
		r := request("/media/import", "bucket=catalog", "demo")
		w := httptest.NewRecorder()
		method(w, r)
		if w.Code != 403 {
			t.Error("read-only account reached save")
		}
		r = request("/media/import", "bucket=catalog", "general")
		r.Header.Set("Origin", "https://untrusted.invalid")
		w = httptest.NewRecorder()
		method(w, r)
		if w.Code != 403 {
			t.Error("cross-origin save accepted")
		}
	}
	var savedID string
	for n := 0; n < 2; n++ {
		r := request("/media/import", "locale=ja&provider=open_library&external_id=OL1W&media_type=book&bucket=catalog", "general")
		w := httptest.NewRecorder()
		mh.ImportSubmit(w, r)
		var result struct {
			EntryID, Bucket, URL string
			Existing             bool
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || w.Header().Get("Location") != "" || result.Bucket != "catalog" || result.Existing != (n == 1) || !strings.HasPrefix(result.URL, "/ja/library/") {
			t.Fatalf("incorrect dialog response: %d %+v", w.Code, result)
		}
		if n == 0 {
			savedID = result.EntryID
		} else if result.EntryID != savedID {
			t.Error("repeat save created another entry")
		}
	}
	r := request("/library/attach", "locale=en&media_record_id="+record.ID.Hex()+"&bucket=wishlist", "general")
	w := httptest.NewRecorder()
	lh.AttachSubmit(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"bucket":"wishlist"`) || w.Header().Get("Location") != "" {
		t.Fatalf("local dialog attach failed: %d %s", w.Code, w.Body.String())
	}
	preview := httptest.NewRequest("GET", "/en/search/details?fragment=1&media_record_id="+record.ID.Hex(), nil)
	preview = preview.WithContext(middleware.WithUser(preview.Context(), &domainauth.User{ID: owner, Role: "general"}))
	w = httptest.NewRecorder()
	mh.Preview(w, preview)
	if w.Code != 200 || strings.Count(w.Body.String(), "Open saved item") != 2 || strings.Contains(w.Body.String(), `<form`) || strings.Contains(w.Body.String(), otherCopy.ID.Hex()) {
		t.Fatalf("preview did not reflect personal saves: %d %s", w.Code, w.Body.String())
	}
	// Normal forms remain functional when JavaScript is unavailable.
	r = request("/library/attach", "locale=en&media_record_id="+record.ID.Hex()+"&bucket=wishlist", "general")
	r.Header.Del("X-Search-Dialog")
	w = httptest.NewRecorder()
	lh.AttachSubmit(w, r)
	if w.Code != 302 || !strings.Contains(w.Header().Get("Location"), "/en/library/") {
		t.Error("normal form fallback no longer redirects to saved copy")
	}
}
