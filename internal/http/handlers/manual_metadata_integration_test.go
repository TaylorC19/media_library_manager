package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	domainauth "media_library_manager/internal/domain/auth"
	domainlib "media_library_manager/internal/domain/library"
	"media_library_manager/internal/http/middleware"
	"media_library_manager/internal/repository"
	libsvc "media_library_manager/internal/service/library"
	"media_library_manager/internal/views"
)

func TestManualMetadataCreateEditSearchAndClear(t *testing.T) {
	database := regressionDatabase(t)
	ctx := context.Background()
	owner := primitive.NewObjectID()
	media := repository.NewMediaRecordsRepository(database)
	entries := repository.NewLibraryEntriesRepository(database)
	service := libsvc.NewService(media, entries)
	renderer, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewLibraryHandler(renderer, service)
	for _, kind := range domainlib.MediaTypes {
		year := 2020
		form := libsvc.ManualCreateForm{Title: "Manual " + kind, MediaType: kind, Bucket: "catalog", Year: &year, ImageURL: "https://example.com/cover.jpg", Summary: "Description <script>plain text</script>", Creators: "First Creator\r\nFamily, Given\nfirst creator\nSecond Creator", Notes: "Private copy note", Tags: "favorite"}
		id, _, err := service.CreateManual(ctx, owner, form)
		if err != nil {
			t.Fatal(err)
		}
		detail, err := service.GetEntryDetail(ctx, owner, id)
		if err != nil {
			t.Fatal(err)
		}
		if detail.Media.ImageURL == nil || *detail.Media.ImageURL != form.ImageURL || detail.Media.Summary == nil || *detail.Media.Summary != form.Summary || libsvc.ManualCreatorText(detail.Media) != "First Creator\nFamily, Given\nSecond Creator" {
			t.Fatal("manual metadata was not stored")
		}
		found, err := service.ListBucket(ctx, owner, "catalog", libsvc.ListBucketFilters{MediaType: kind, Query: "Second Creator"}, 1, 10)
		if err != nil || found.TotalItems != 1 {
			t.Fatal("manual creators are not searchable")
		}
		request := func(method string, fields url.Values) *http.Request {
			r := httptest.NewRequest(method, "/en/library/"+id.Hex()+"/edit", strings.NewReader(fields.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			route := chi.NewRouteContext()
			route.URLParams.Add("entryId", id.Hex())
			return r.WithContext(context.WithValue(middleware.WithUser(r.Context(), &domainauth.User{ID: owner, Role: "general"}), chi.RouteCtxKey, route))
		}
		w := httptest.NewRecorder()
		handler.EditForm(w, request("GET", nil))
		html := w.Body.String()
		for _, value := range []string{`name="image_url"`, `name="description"`, `name="creators"`, form.ImageURL, "Family, Given", "Description &lt;script&gt;plain text&lt;/script&gt;"} {
			if !strings.Contains(html, value) {
				t.Errorf("edit missing %q", value)
			}
		}
		w = httptest.NewRecorder()
		handler.Detail(w, request("GET", nil))
		if !strings.Contains(w.Body.String(), form.ImageURL) || !strings.Contains(w.Body.String(), "Second Creator") || strings.Contains(w.Body.String(), "<script>plain text") {
			t.Error("manual details are missing or unsafe")
		}
		// Failed validation must keep entered fields in the form and leave storage intact.
		fields := url.Values{"locale": {"en"}, "title": {"Rejected title"}, "media_type": {kind}, "bucket": {"catalog"}, "year": {"2020"}, "image_url": {"javascript:alert(1)"}, "description": {"Still entered description"}, "creators": {"Still Entered Creator"}, "notes": {"Private copy note"}, "tags": {"favorite"}}
		w = httptest.NewRecorder()
		handler.UpdateSubmit(w, request("POST", fields))
		if !strings.Contains(w.Body.String(), "Still entered description") || !strings.Contains(w.Body.String(), "Still Entered Creator") {
			t.Error("validation lost entered metadata")
		}
		current, _ := service.GetEntryDetail(ctx, owner, id)
		if current.Media.Title != form.Title {
			t.Error("invalid edit partially changed metadata")
		}
		fields.Set("title", form.Title)
		fields.Set("image_url", "https://example.com/new-cover.jpg")
		w = httptest.NewRecorder()
		handler.UpdateSubmit(w, request("POST", fields))
		if w.Code != 302 {
			t.Fatalf("valid edit failed: %d %s", w.Code, w.Body.String())
		}
		current, _ = service.GetEntryDetail(ctx, owner, id)
		if *current.Media.ImageURL != "https://example.com/new-cover.jpg" || *current.Media.Summary != "Still entered description" || libsvc.ManualCreatorText(current.Media) != "Still Entered Creator" || *current.Entry.Notes != "Private copy note" {
			t.Error("edit lost manual metadata or copy details")
		}
		fields.Set("image_url", "")
		fields.Set("description", "")
		fields.Set("creators", "")
		w = httptest.NewRecorder()
		handler.UpdateSubmit(w, request("POST", fields))
		current, _ = service.GetEntryDetail(ctx, owner, id)
		if w.Code != 302 || current.Media.ImageURL != nil || current.Media.Summary != nil || libsvc.ManualCreatorText(current.Media) != "" {
			t.Error("clearing optional metadata did not persist")
		}
	}
}

func TestManualMetadataSharedCopiesAndProviderFieldsAreProtected(t *testing.T) {
	database := regressionDatabase(t)
	ctx := context.Background()
	owner, other := primitive.NewObjectID(), primitive.NewObjectID()
	media := repository.NewMediaRecordsRepository(database)
	entries := repository.NewLibraryEntriesRepository(database)
	service := libsvc.NewService(media, entries)
	id, _, err := service.CreateManual(ctx, owner, libsvc.ManualCreateForm{Title: "Shared original", MediaType: "book", Bucket: "catalog", Format: "paperback", Summary: "Original description", Creators: "Original Author", Notes: "Owner note"})
	if err != nil {
		t.Fatal(err)
	}
	original, _ := service.GetEntryDetail(ctx, owner, id)
	otherCopy := &domainlib.LibraryEntry{UserID: other, MediaRecordID: original.Media.ID, MediaType: "book", Bucket: "catalog"}
	if err := entries.Insert(ctx, otherCopy); err != nil {
		t.Fatal(err)
	}
	conflict := &domainlib.LibraryEntry{UserID: owner, MediaRecordID: original.Media.ID, MediaType: "book", Bucket: "catalog", Format: ptrString("hardcover")}
	if err := entries.Insert(ctx, conflict); err != nil {
		t.Fatal(err)
	}
	update := libsvc.ManualUpdateForm{Title: "My corrected title", MediaType: "book", Bucket: "catalog", Format: "hardcover", Summary: "My description", Creators: "My Author", Notes: "Owner note"}
	if _, err := service.UpdateEntry(ctx, owner, id, update); !errors.Is(err, libsvc.ErrValidation) {
		t.Error("copy conflict should fail before metadata changes")
	}
	unchanged, _ := media.FindByID(ctx, original.Media.ID)
	if unchanged.Title != "Shared original" {
		t.Error("failed copy edit changed metadata")
	}
	update.Format = "paperback"
	if _, err := service.UpdateEntry(ctx, owner, id, update); err != nil {
		t.Fatal(err)
	}
	mine, _ := service.GetEntryDetail(ctx, owner, id)
	theirs, _ := service.GetEntryDetail(ctx, other, otherCopy.ID)
	if mine.Media.ID == theirs.Media.ID || mine.Media.Title != "My corrected title" || theirs.Media.Title != "Shared original" || libsvc.ManualCreatorText(theirs.Media) != "Original Author" {
		t.Error("editing a shared manual record changed another user's metadata")
	}
	if *mine.Entry.Notes != "Owner note" || mine.Entry.ID != id {
		t.Error("copy-on-write lost entry identity or notes")
	}
	provider := &domainlib.MediaRecord{Source: "provider", Title: "Provider title", MediaType: "book", ImageURL: ptrString("https://example.com/provider.jpg"), Summary: ptrString("Provider summary"), Details: bson.M{"authors": []string{"Provider Author"}}}
	if err := media.Insert(ctx, provider); err != nil {
		t.Fatal(err)
	}
	saved := &domainlib.LibraryEntry{UserID: owner, MediaRecordID: provider.ID, Bucket: "catalog", MediaType: "book"}
	if err := entries.Insert(ctx, saved); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateEntry(ctx, owner, saved.ID, libsvc.ManualUpdateForm{Bucket: "catalog", ImageURL: "javascript:alert(1)", Summary: "forged", Creators: "forged", Title: "forged", Notes: "Only copy notes change"}); err != nil {
		t.Fatal(err)
	}
	stored, _ := media.FindByID(ctx, provider.ID)
	if stored.Title != "Provider title" || *stored.ImageURL != *provider.ImageURL || *stored.Summary != "Provider summary" || libsvc.ManualCreatorText(stored) != "Provider Author" {
		t.Error("provider metadata accepted forged manual fields")
	}
	if _, err := service.UpdateEntry(ctx, other, id, update); !errors.Is(err, libsvc.ErrNotFound) {
		t.Error("another user could edit my manual item")
	}
}

func ptrString(value string) *string { return &value }
