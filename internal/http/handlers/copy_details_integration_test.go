package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
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

func TestDialogCopyDetailsAndRelevantFormats(t *testing.T) {
	database := regressionDatabase(t)
	ctx := context.Background()
	owner := primitive.NewObjectID()
	media := repository.NewMediaRecordsRepository(database)
	entries := repository.NewLibraryEntriesRepository(database)
	if err := entries.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	render, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	service := libsvc.NewService(media, entries)
	mh := NewMediaHandler(render, mediasvc.NewService(config.Config{}, media, entries, nil, reliability.NewService(config.Config{}), nil))
	lh := NewLibraryHandler(render, service)
	record := &domainlib.MediaRecord{Title: "Fixture book", Source: "provider", MediaType: "book", ProviderRefs: bson.M{"openLibrary": bson.M{"workKey": "OLCOPYW"}}}
	if err := media.Insert(ctx, record); err != nil {
		t.Fatal(err)
	}
	post := func(action string, fields url.Values) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", action, strings.NewReader(fields.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("X-Search-Dialog", "1")
		r = r.WithContext(middleware.WithUser(r.Context(), &domainauth.User{ID: owner, Role: "general"}))
		w := httptest.NewRecorder()
		if action == "/media/import" {
			mh.ImportSubmit(w, r)
		} else {
			lh.AttachSubmit(w, r)
		}
		return w
	}
	for _, action := range []string{"/media/import", "/library/attach"} {
		bucket := "catalog"
		if action == "/library/attach" {
			bucket = "wishlist"
		}
		fields := url.Values{"locale": {"en"}, "media_record_id": {record.ID.Hex()}, "provider": {"open_library"}, "external_id": {"OLCOPYW"}, "media_type": {"book"}, "bucket": {bucket}, "format": {"paperback"}, "notes": {" Original copy notes "}}
		var firstID primitive.ObjectID
		for attempt := 0; attempt < 2; attempt++ {
			if attempt == 1 {
				fields.Set("notes", "Must not overwrite original")
			}
			w := post(action, fields)
			var result struct {
				EntryID, Format string
				Existing        bool
			}
			if w.Code != 200 {
				t.Fatalf("save failed: %d %s", w.Code, w.Body.String())
			}
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			id, err := primitive.ObjectIDFromHex(result.EntryID)
			if err != nil {
				t.Fatal(err)
			}
			if attempt == 0 {
				firstID = id
			} else if id != firstID || !result.Existing {
				t.Error("same-format save must reuse the existing copy")
			}
			copy, err := entries.FindByIDAndUser(ctx, id, owner)
			if err != nil || copy == nil || copy.Format == nil || *copy.Format != "paperback" || copy.Notes == nil || *copy.Notes != "Original copy notes" || result.Format != "paperback" {
				t.Fatal("copy details were lost or overwritten")
			}
		}
		fields.Set("format", "hardcover")
		fields.Set("notes", "Different copy notes")
		w := post(action, fields)
		if w.Code != 200 {
			t.Fatal(w.Body.String())
		}
		copies, err := entries.FindByUserAndMediaRecordIDs(ctx, owner, []primitive.ObjectID{record.ID})
		if err != nil {
			t.Fatal(err)
		}
		inBucket := 0
		for _, copy := range copies {
			if copy.Bucket == bucket {
				inBucket++
			}
		}
		if inBucket != 2 {
			t.Error("different format must create its own saved copy")
		}
		fields.Set("format", "ps5")
		if w := post(action, fields); w.Code != 400 {
			t.Error("unrelated format accepted by server")
		}
	}
	movie := &domainlib.MediaRecord{Title: "Legacy movie", Source: "provider", MediaType: "movie"}
	if err := media.Insert(ctx, movie); err != nil {
		t.Fatal(err)
	}
	legacy := "hardcover"
	entry := &domainlib.LibraryEntry{UserID: owner, MediaRecordID: movie.ID, MediaType: "movie", Bucket: "catalog", Format: &legacy}
	if err := entries.Insert(ctx, entry); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateEntry(ctx, owner, entry.ID, libsvc.ManualUpdateForm{Bucket: "catalog", Format: legacy, Notes: "Preserved legacy format"}); err != nil {
		t.Fatalf("existing legacy format must remain editable: %v", err)
	}
	if _, err := service.UpdateEntry(ctx, owner, entry.ID, libsvc.ManualUpdateForm{Bucket: "catalog", Format: "vinyl"}); !errors.Is(err, libsvc.ErrValidation) {
		t.Error("legacy exception must not permit another unrelated format")
	}
	r := httptest.NewRequest("GET", "/en/library/"+entry.ID.Hex()+"/edit", nil)
	route := chi.NewRouteContext()
	route.URLParams.Add("entryId", entry.ID.Hex())
	r = r.WithContext(context.WithValue(middleware.WithUser(r.Context(), &domainauth.User{ID: owner, Role: "general"}), chi.RouteCtxKey, route))
	w := httptest.NewRecorder()
	lh.EditForm(w, r)
	html := w.Body.String()
	start := strings.Index(html, `<select name="format"`)
	end := strings.Index(html[start:], "</select>")
	options := html[start : start+end]
	if w.Code != 200 || !strings.Contains(options, `value="hardcover" selected`) || !strings.Contains(options, `value="dvd"`) || strings.Contains(options, `value="ps5"`) {
		t.Error("edit form must show relevant formats plus the saved legacy format")
	}
	if _, _, err := service.CreateManual(ctx, owner, libsvc.ManualCreateForm{Title: "Invalid new movie", MediaType: "movie", Bucket: "catalog", Format: "hardcover"}); !errors.Is(err, libsvc.ErrValidation) {
		t.Error("new manual copies must reject irrelevant formats")
	}
}
