package app

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"media_library_manager/internal/config"
)

func TestProductionAuthenticationAndSettingsFlow(t *testing.T) {
	t.Setenv("VERCEL", "")
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		uri = "mongodb://127.0.0.1:27017"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(context.Background())
	if err := client.Ping(ctx, nil); err != nil {
		if os.Getenv("REQUIRE_MONGO_TESTS") == "1" {
			t.Fatal(err)
		}
		t.Skip("requires test MongoDB")
	}
	database := client.Database("mlm_release_fixture_" + primitive.NewObjectID().Hex())
	t.Cleanup(func() { _ = database.Drop(context.Background()) })
	cfg := config.Config{Env: "production", PublicOrigin: "https://library.example", DefaultLocale: "en", MongoURI: uri, MongoDatabase: database.Name(), SessionCookieName: "__Host-mlm_session", SessionCookieSecure: true, SessionTTLHours: 1, ProviderKeysEncryptionKey: base64.StdEncoding.EncodeToString(make([]byte, 32)), EnsureIndexes: true}
	application, err := New(cfg)
	if err != nil {
		if os.Getenv("REQUIRE_MONGO_TESTS") == "1" {
			t.Fatal(err)
		}
		t.Skip("requires test replica set")
	}
	defer application.Close(context.Background())
	if _, err := application.Auth.CreateAccount(context.Background(), "fixture-admin", "synthetic-fixture-password", "", "admin"); err != nil {
		t.Fatal(err)
	}
	request := func(method, path, origin string, fields url.Values, cookie *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, cfg.PublicOrigin+path, strings.NewReader(fields.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		application.Router.ServeHTTP(w, r)
		return w
	}
	fields := url.Values{"username": {"fixture-admin"}, "password": {"synthetic-fixture-password"}}
	if w := request("POST", "/en/login", "https://foreign.example", fields, nil); w.Code != 403 {
		t.Fatal("login origin not protected")
	}
	if w := request("GET", "/en/register", "", nil, nil); w.Code != 404 {
		t.Fatal("public signup not disabled")
	}
	w := request("POST", "/en/login", cfg.PublicOrigin, fields, nil)
	if w.Code != 302 {
		t.Fatalf("login failed: %d", w.Code)
	}
	var cookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == cfg.SessionCookieName {
			cookie = c
		}
	}
	if cookie == nil || !cookie.Secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" || cookie.Domain != "" {
		t.Fatal("session cookie policy not enforced")
	}
	for _, path := range []string{"/en/", "/en/settings", "/en/admin/users"} {
		w = request("GET", path, "", nil, cookie)
		if w.Code != 200 || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
			t.Fatalf("private page %s: %d", path, w.Code)
		}
	}
	keys := url.Values{"tmdb": {"synthetic-provider-fixture"}}
	w = request("POST", "/en/settings/providers", cfg.PublicOrigin, keys, cookie)
	if w.Code != 303 {
		t.Fatalf("provider settings save: %d", w.Code)
	}
	w = request("GET", "/en/settings", "", nil, cookie)
	if w.Code != 200 || strings.Contains(w.Body.String(), "synthetic-provider-fixture") {
		t.Fatal("provider secret exposed in settings")
	}
	for _, path := range []string{"/settings/backup/export", "/settings/backup/import"} {
		w = request("POST", path, cfg.PublicOrigin, nil, cookie)
		if w.Code != 403 {
			t.Fatalf("production maintenance accessible: %d", w.Code)
		}
	}
	w = request("POST", "/en/logout", cfg.PublicOrigin, nil, cookie)
	if w.Code != 302 || w.Header().Get("Clear-Site-Data") == "" {
		t.Fatal("logout privacy controls missing")
	}
	w = request("GET", "/en/settings", "", nil, cookie)
	if w.Code != 302 {
		t.Fatal("logged-out session remained usable")
	}
}
