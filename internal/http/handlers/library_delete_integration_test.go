package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	domainauth "media_library_manager/internal/domain/auth"
	domainlib "media_library_manager/internal/domain/library"
	"media_library_manager/internal/http/middleware"
	"media_library_manager/internal/repository"
	libsvc "media_library_manager/internal/service/library"
	"media_library_manager/internal/views"
)

func TestLibraryDeleteRequiresConfirmationAndPreservesOtherCopies(t *testing.T) {
	database := regressionDatabase(t)
	ctx := context.Background()
	owner, other := primitive.NewObjectID(), primitive.NewObjectID()
	media := repository.NewMediaRecordsRepository(database)
	entries := repository.NewLibraryEntriesRepository(database)
	renderer, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewLibraryHandler(renderer, libsvc.NewService(media, entries))
	for _, bucket := range []string{"catalog", "wishlist"} {
		record := &domainlib.MediaRecord{Title: "Delete only this <script>copy</script>", Source: "manual", MediaType: "book"}
		if err := media.Insert(ctx, record); err != nil {
			t.Fatal(err)
		}
		format, notes := "paperback", "Keep my notes until I confirm"
		own := &domainlib.LibraryEntry{UserID: owner, MediaRecordID: record.ID, Bucket: bucket, MediaType: "book", Format: &format, Notes: &notes}
		another := &domainlib.LibraryEntry{UserID: other, MediaRecordID: record.ID, Bucket: bucket, MediaType: "book"}
		for _, entry := range []*domainlib.LibraryEntry{own, another} {
			if err := entries.Insert(ctx, entry); err != nil {
				t.Fatal(err)
			}
		}
		request := func(method, locale, body, role, origin string, userID primitive.ObjectID) *httptest.ResponseRecorder {
			path := "/" + locale + "/library/" + own.ID.Hex() + "/delete"
			if method == "POST" {
				path = "/library/" + own.ID.Hex() + "/delete"
			}
			r := httptest.NewRequest(method, path, strings.NewReader(body))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if origin != "" {
				r.Header.Set("Origin", origin)
			}
			route := chi.NewRouteContext()
			route.URLParams.Add("entryId", own.ID.Hex())
			r = r.WithContext(context.WithValue(middleware.WithUser(r.Context(), &domainauth.User{ID: userID, Role: role}), chi.RouteCtxKey, route))
			w := httptest.NewRecorder()
			action := handler.DeleteConfirmation
			if method == "POST" {
				action = handler.DeleteSubmit
			}
			middleware.Locale("en")(http.HandlerFunc(action)).ServeHTTP(w, r)
			return w
		}
		for _, locale := range []string{"en", "ja"} {
			w := request("GET", locale, "", "general", "", owner)
			html := w.Body.String()
			if w.Code != 200 || !strings.Contains(html, "Delete only this &lt;script&gt;copy&lt;/script&gt;") || !strings.Contains(html, own.ID.Hex()) || strings.Contains(html, "<script>copy") {
				t.Fatalf("confirmation did not safely name the selected item: %d %s", w.Code, html)
			}
			if !strings.Contains(html, `name="confirm" value="delete"`) || !strings.Contains(html, "/"+locale+"/library/"+own.ID.Hex()+`"`) {
				t.Error("confirmation and cancel must target the same saved copy")
			}
			if w.Header().Get("Cache-Control") != "private, no-store" {
				t.Error("confirmation must not be cached")
			}
		}
		for _, test := range []struct {
			role, body, origin string
			user               primitive.ObjectID
			want               int
		}{
			{"general", "locale=en", "", owner, 400},
			{"general", "locale=en&confirm=wrong", "", owner, 400},
			{"demo", "locale=en&confirm=delete", "", owner, 403},
			{"general", "locale=en&confirm=delete", "https://untrusted.invalid", owner, 403},
			{"admin", "locale=en&confirm=delete", "", other, 404},
		} {
			w := request("POST", "en", test.body, test.role, test.origin, test.user)
			if w.Code != test.want {
				t.Errorf("unconfirmed/forbidden delete got %d want %d", w.Code, test.want)
			}
			saved, err := entries.FindByIDAndUser(ctx, own.ID, owner)
			if err != nil || saved == nil || saved.Notes == nil || *saved.Notes != notes {
				t.Fatal("unconfirmed delete changed the saved copy")
			}
		}
		if w := request("GET", "en", "", "general", "", other); w.Code != 404 {
			t.Error("confirmation exposed another user's copy")
		}
		w := request("POST", "en", "locale=en&confirm=delete", "general", "", owner)
		if w.Code != 302 || w.Header().Get("Location") != libraryBucketPath("en", bucket) {
			t.Fatalf("confirmed delete returned to wrong bucket: %d %s", w.Code, w.Header().Get("Location"))
		}
		saved, err := entries.FindByIDAndUser(ctx, own.ID, owner)
		if err != nil || saved != nil {
			t.Error("confirmed delete did not remove selected entry")
		}
		saved, err = entries.FindByIDAndUser(ctx, another.ID, other)
		if err != nil || saved == nil {
			t.Error("delete affected another user's copy")
		}
		rec, err := media.FindByID(ctx, record.ID)
		if err != nil || rec == nil {
			t.Error("delete removed metadata still used by another copy")
		}
	}
}
