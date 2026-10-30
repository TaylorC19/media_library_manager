package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	domainauth "media_library_manager/internal/domain/auth"
	"media_library_manager/internal/http/middleware"
	"media_library_manager/internal/providerkeys"
	"media_library_manager/internal/repository"
	"media_library_manager/internal/service/adminusers"
	"media_library_manager/internal/views"
)

func TestConfirmedAdminKeyClearIsPrivateAndPreciselyScoped(t *testing.T) {
	database := regressionDatabase(t)
	ctx := context.Background()
	users := repository.NewUsersRepository(database)
	actor := &domainauth.User{Username: "qa-admin", Role: "admin"}
	target := &domainauth.User{Username: "qa-target", Role: "general"}
	other := &domainauth.User{Username: "qa-other", Role: "general"}
	for _, user := range []*domainauth.User{actor, target, other} {
		if err := users.Create(ctx, user); err != nil {
			t.Fatal(err)
		}
	}
	store := providerkeys.NewStore(t.TempDir())
	if err := store.Update(target.ID, providerkeys.Keys{"tmdb": "target-private-key", "rawg": "keep-target-rawg"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Update(other.ID, providerkeys.Keys{"tmdb": "keep-other-key"}); err != nil {
		t.Fatal(err)
	}
	renderer, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewUserKeysHandler(renderer, adminusers.NewKeyService(adminusers.NewService(users, nil), store))
	request := func(method, path, body string) *http.Request {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		route := chi.NewRouteContext()
		route.URLParams.Add("userId", target.ID.Hex())
		r = r.WithContext(context.WithValue(middleware.WithUser(r.Context(), actor), chi.RouteCtxKey, route))
		return r
	}
	assertPrivate := func(body string) {
		t.Helper()
		for _, secret := range []string{"target-private-key", "keep-target-rawg", "keep-other-key"} {
			if strings.Contains(body, secret) {
				t.Error("key value leaked into response")
			}
		}
	}
	base := "/en/admin/users/" + target.ID.Hex() + "/keys"
	r := request("GET", base+"?q=target&role=general&page=2", "")
	w := httptest.NewRecorder()
	handler.Page(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "qa-target") {
		t.Error("saved-key overview failed")
	}
	assertPrivate(w.Body.String())
	r = request("GET", base+"/clear?provider=tmdb&q=target&role=general&page=2", "")
	w = httptest.NewRecorder()
	handler.Confirm(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `name="confirm" value="clear" required`) || !strings.Contains(w.Body.String(), "TMDB") {
		t.Error("confirmation must show target/provider and require explicit acknowledgement")
	}
	assertPrivate(w.Body.String())
	keys, _ := store.Get(target.ID)
	if keys["tmdb"] != "target-private-key" {
		t.Fatal("GET confirmation cleared a key")
	}
	r = request("POST", base+"/clear", "provider=tmdb")
	w = httptest.NewRecorder()
	handler.Clear(w, r)
	keys, _ = store.Get(target.ID)
	if w.Code != 400 || keys["tmdb"] != "target-private-key" {
		t.Error("missing confirmation must not clear")
	}
	assertPrivate(w.Body.String())
	r = request("POST", base+"/clear", "provider=tmdb&confirm=clear")
	r.Header.Set("Origin", "https://untrusted.invalid")
	w = httptest.NewRecorder()
	handler.Clear(w, r)
	if w.Code != 403 {
		t.Error("cross-origin clear must be rejected")
	}
	if err := users.SetRole(ctx, actor.Username, "general"); err != nil {
		t.Fatal(err)
	}
	r = request("POST", base+"/clear", "provider=tmdb&confirm=clear")
	w = httptest.NewRecorder()
	handler.Clear(w, r)
	if w.Code != 403 {
		t.Error("stale admin session must not clear keys after demotion")
	}
	if err := users.SetRole(ctx, actor.Username, "admin"); err != nil {
		t.Fatal(err)
	}
	r = request("POST", base+"/clear", "provider=tmdb&confirm=clear&q=target&filter_role=general&page=2")
	w = httptest.NewRecorder()
	handler.Clear(w, r)
	if w.Code != 303 || !strings.Contains(w.Header().Get("Location"), "role=general") || !strings.Contains(w.Header().Get("Location"), "q=target") {
		t.Error("successful clear must retain navigation context")
	}
	assertPrivate(w.Body.String())
	keys, _ = store.Get(target.ID)
	otherKeys, _ := store.Get(other.ID)
	if keys["tmdb"] != "" || keys["rawg"] != "keep-target-rawg" || otherKeys["tmdb"] != "keep-other-key" {
		t.Error("clear affected another provider/account")
	}
	if providerkeys.Resolve(providerkeys.WithKeys(ctx, keys), "tmdb", "server-default") != "server-default" {
		t.Error("cleared override must fall back to server configuration")
	}
}
