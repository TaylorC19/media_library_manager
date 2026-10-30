package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
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

func TestMoveWishlistPreservesCopyAndHandlesConflicts(t *testing.T) {
	database := regressionDatabase(t)
	ctx := context.Background()
	media := repository.NewMediaRecordsRepository(database)
	entries := repository.NewLibraryEntriesRepository(database)
	if err := entries.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	renderer, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewLibraryHandler(renderer, libsvc.NewService(media, entries))
	owner := primitive.NewObjectID()
	record := &domainlib.MediaRecord{Title: "Saved film", Source: domainlib.SourceProvider, MediaType: "movie", ProviderRefs: bson.M{"tmdb": bson.M{"id": "42", "kind": "movie"}}, Details: bson.M{"directors": []string{"Director"}}}
	if err := media.Insert(ctx, record); err != nil {
		t.Fatal(err)
	}
	note, barcode, date, format := "Keep\nmy notes", "1234567890123", "2026-10-01", "blu_ray"
	entry := &domainlib.LibraryEntry{UserID: owner, MediaRecordID: record.ID, MediaType: "movie", Bucket: "wishlist", Notes: &note, Barcode: &barcode, PurchaseDate: &date, Format: &format, Tags: []string{"favorite", "gift"}}
	if err := entries.Insert(ctx, entry); err != nil {
		t.Fatal(err)
	}
	raw := func(id primitive.ObjectID) bson.M {
		t.Helper()
		var value bson.M
		if err := database.Collection("library_entries").FindOne(ctx, bson.M{"_id": id}).Decode(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	before := raw(entry.ID)
	submit := func(id primitive.ObjectID, user *domainauth.User, origin string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/library/"+id.Hex()+"/move-to-collection", strings.NewReader("locale=ja&notes=forged&bucket=wishlist&userId=forged"))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		route := chi.NewRouteContext()
		route.URLParams.Add("entryId", id.Hex())
		r = r.WithContext(context.WithValue(middleware.WithUser(r.Context(), user), chi.RouteCtxKey, route))
		w := httptest.NewRecorder()
		handler.MoveToCollectionSubmit(w, r)
		return w
	}
	for _, test := range []struct {
		user   *domainauth.User
		origin string
		status int
	}{
		{nil, "", 401},
		{&domainauth.User{ID: owner, Role: "demo"}, "", 403},
		{&domainauth.User{ID: owner, Role: "general"}, "https://untrusted.invalid", 403},
		{&domainauth.User{ID: primitive.NewObjectID(), Role: "admin"}, "", 404},
	} {
		if w := submit(entry.ID, test.user, test.origin); w.Code != test.status {
			t.Errorf("denied move: got %d want %d", w.Code, test.status)
		}
		if !reflect.DeepEqual(before, raw(entry.ID)) {
			t.Fatal("denied move modified copy")
		}
	}
	for _, role := range []string{"general", "admin"} {
		// The second request repeats the completed move and must remain harmless.
		w := submit(entry.ID, &domainauth.User{ID: owner, Role: role}, "")
		if w.Code != http.StatusFound || w.Header().Get("Location") != "/ja/library/"+entry.ID.Hex() {
			t.Fatalf("move redirect: %d %s", w.Code, w.Header().Get("Location"))
		}
	}
	after := raw(entry.ID)
	if after["bucket"] != "catalog" {
		t.Fatal("item was not moved")
	}
	delete(before, "bucket")
	delete(before, "updatedAt")
	delete(after, "bucket")
	delete(after, "updatedAt")
	if !reflect.DeepEqual(before, after) {
		t.Fatal("move changed copy fields or identity")
	}
	savedMedia, err := media.FindByID(ctx, record.ID)
	if err != nil || savedMedia.Title != record.Title || !reflect.DeepEqual(savedMedia.ProviderRefs, record.ProviderRefs) {
		t.Fatal("move changed shared media")
	}
	count, err := entries.CountByMediaRecordID(ctx, record.ID)
	if err != nil || count != 1 {
		t.Fatal("move duplicated the saved item")
	}
	// A matching Collection copy must never be silently replaced or merged.
	wishNote := "Wishlist-only notes"
	conflict := &domainlib.LibraryEntry{UserID: owner, MediaRecordID: record.ID, MediaType: "movie", Bucket: "wishlist", Notes: &wishNote, Format: &format}
	if err := entries.Insert(ctx, conflict); err != nil {
		t.Fatal(err)
	}
	existingBefore, wishBefore := raw(entry.ID), raw(conflict.ID)
	w := submit(conflict.ID, &domainauth.User{ID: owner, Role: "general"}, "")
	if w.Code != 302 {
		t.Fatalf("conflict must return to detail with explanatory flash: %d", w.Code)
	}
	found := false
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "flash" {
			flash := middleware.DecodeFlash(cookie.Value)
			found = flash != nil && flash.Message == "library.errors.moveConflict"
		}
	}
	if !found {
		t.Error("conflict explanation missing")
	}
	if !reflect.DeepEqual(existingBefore, raw(entry.ID)) || !reflect.DeepEqual(wishBefore, raw(conflict.ID)) {
		t.Fatal("conflict changed existing copy details")
	}
	// The form is visible only for a writable Wishlist entry, in either locale.
	for _, locale := range []string{"en", "ja"} {
		for _, bucket := range []string{"catalog", "wishlist"} {
			for _, role := range []string{"general", "demo"} {
				var out strings.Builder
				err := renderer.Render(&out, "pages/library_detail.content", map[string]any{"Locale": locale, "User": &domainauth.User{Role: role}, "Detail": &libsvc.EntryWithMedia{Entry: &domainlib.LibraryEntry{ID: entry.ID, Bucket: bucket}, Media: record}})
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(out.String(), "/move-to-collection") != (bucket == "wishlist" && role == "general") {
					t.Error("move form visibility incorrect")
				}
			}
		}
	}
}
