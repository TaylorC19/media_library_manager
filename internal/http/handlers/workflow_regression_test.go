package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"media_library_manager/internal/config"
	domainauth "media_library_manager/internal/domain/auth"
	domainlib "media_library_manager/internal/domain/library"
	"media_library_manager/internal/http/middleware"
	"media_library_manager/internal/repository"
	authsvc "media_library_manager/internal/service/auth"
	libsvc "media_library_manager/internal/service/library"
	mediasvc "media_library_manager/internal/service/media"
	"media_library_manager/internal/views"
)

func regressionDatabase(t *testing.T) *mongo.Database {
	t.Helper()
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		uri = "mongodb://127.0.0.1:27017"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Skip("Mongo test instance unavailable")
	}
	if err := client.Ping(ctx, nil); err != nil {
		client.Disconnect(context.Background())
		t.Skip("Mongo test instance unavailable")
	}
	database := client.Database("mlm_qa_regression_" + primitive.NewObjectID().Hex())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		database.Drop(ctx)
		client.Disconnect(ctx)
	})
	return database
}

func TestRegistrationCreatesGeneralUserAndRoleChangesApplyToExistingSession(t *testing.T) {
	database := regressionDatabase(t)
	users := repository.NewUsersRepository(database)
	sessions := repository.NewSessionsRepository(database)
	ctx := context.Background()
	if err := users.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	if err := sessions.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	service := authsvc.NewService(users, sessions, time.Hour)
	user, token, err := service.Register(ctx, "qa-role-test", "temporary-test-password")
	if err != nil {
		t.Fatal(err)
	}
	if user.ID.IsZero() || user.EffectiveRole() != domainauth.RoleGeneral {
		t.Fatal("registration must create a general account with a usable ID")
	}
	resolved, _, err := service.ResolveSession(ctx, token)
	if err != nil || resolved == nil || resolved.ID != user.ID {
		t.Fatal("registration session must resolve")
	}
	if err := users.SetRole(ctx, user.Username, domainauth.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	resolved, _, err = service.ResolveSession(ctx, token)
	if err != nil || resolved == nil || !resolved.IsAdmin() {
		t.Fatal("role changes must apply without creating another session")
	}
}

func TestWishlistDeleteRedirectAndBarcodeImportPersistence(t *testing.T) {
	database := regressionDatabase(t)
	ctx := context.Background()
	media := repository.NewMediaRecordsRepository(database)
	entries := repository.NewLibraryEntriesRepository(database)
	if err := media.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	if err := entries.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	owner := primitive.NewObjectID()
	record := &domainlib.MediaRecord{Title: "QA test book", MediaType: "book", Source: domainlib.SourceProvider, ProviderRefs: bson.M{"openLibrary": bson.M{"workKey": "OL1W"}}}
	if err := media.Insert(ctx, record); err != nil {
		t.Fatal(err)
	}
	service := mediasvc.NewService(config.Config{}, media, entries, nil, nil, nil)
	outcome, _, err := service.Import(ctx, owner, mediasvc.ImportForm{Provider: "open_library", ExternalID: "OL1W", MediaType: "book", Bucket: "wishlist", Barcode: "9781234567897"})
	if err != nil {
		t.Fatal(err)
	}
	matches, err := entries.FindByUserAndBarcode(ctx, owner, "9781234567897", nil)
	if err != nil || len(matches) != 1 {
		t.Fatal("scanned barcode must be persisted and locally searchable")
	}
	other, err := entries.FindByUserAndBarcode(ctx, primitive.NewObjectID(), "9781234567897", nil)
	if err != nil || len(other) != 0 {
		t.Fatal("saved barcode lookup must stay private to its owner")
	}
	renderer, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewLibraryHandler(renderer, libsvc.NewService(media, entries))
	r := httptest.NewRequest("POST", "/library/"+outcome.LibraryEntry.ID.Hex()+"/delete", strings.NewReader("confirm=delete"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	route := chi.NewRouteContext()
	route.URLParams.Add("entryId", outcome.LibraryEntry.ID.Hex())
	r = r.WithContext(context.WithValue(middleware.WithUser(r.Context(), &domainauth.User{ID: owner, Role: domainauth.RoleGeneral}), chi.RouteCtxKey, route))
	w := httptest.NewRecorder()
	handler.DeleteSubmit(w, r)
	if w.Code != http.StatusFound || w.Header().Get("Location") != "/en/wishlist" {
		t.Fatalf("wrong delete redirect: %d %s", w.Code, w.Header().Get("Location"))
	}
}
