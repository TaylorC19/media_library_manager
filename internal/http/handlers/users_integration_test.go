package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
	domainauth "media_library_manager/internal/domain/auth"
	"media_library_manager/internal/http/middleware"
	"media_library_manager/internal/repository"
	"media_library_manager/internal/service/adminusers"
	authsvc "media_library_manager/internal/service/auth"
	"media_library_manager/internal/views"
)

func TestAdminAccountManagementEndToEnd(t *testing.T) {
	database := regressionDatabase(t)
	ctx := context.Background()
	users := repository.NewUsersRepository(database)
	sessions := repository.NewSessionsRepository(database)
	if err := users.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	accounts := authsvc.NewService(users, sessions, time.Hour)
	actor, err := accounts.CreateAccount(ctx, "qa-admin", "fixture-password", "QA Admin", domainauth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	manager := adminusers.NewService(users, accounts)
	renderer, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewUsersHandler(renderer, manager)
	request := func(method, path, body string) *http.Request {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return r.WithContext(middleware.WithUser(r.Context(), actor))
	}
	r := request("POST", "/en/admin/users", "username=qa.demo&display_name=QA+Demo&password=temporary-new-password&role=demo")
	w := httptest.NewRecorder()
	handler.Create(w, r)
	if w.Code != 303 {
		t.Fatalf("create status %d", w.Code)
	}
	created, err := users.FindByUsername(ctx, "qa.demo")
	if err != nil || created == nil || created.Role != "demo" {
		t.Fatal("admin-created account missing or wrong role")
	}
	if bcrypt.CompareHashAndPassword([]byte(created.PasswordHash), []byte("temporary-new-password")) != nil {
		t.Error("new password not stored correctly")
	}
	sessionCount, err := database.Collection("sessions").CountDocuments(ctx, bson.M{})
	if err != nil || sessionCount != 0 {
		t.Error("admin account creation must not issue a new session")
	}
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name != "flash" {
			t.Error("account creation changed the admin's session cookie")
		}
	}
	managed, count, err := users.ListManaged(ctx, ".", "", 999, 20)
	if err != nil || count != 1 || len(managed) != 1 || managed[0].PasswordHash != "" {
		t.Error("managed account listing must support literal search, clamp page, and exclude hashes")
	}
	r = request("GET", "/en/admin/users?q=qa.demo", "")
	w = httptest.NewRecorder()
	handler.Page(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "qa.demo") || strings.Contains(w.Body.String(), created.PasswordHash) {
		t.Error("admin listing failed or exposed secret fields")
	}
	route := chi.NewRouteContext()
	route.URLParams.Add("userId", created.ID.Hex())
	r = request("POST", "/en/admin/users/"+created.ID.Hex()+"/role", "role=general")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, route))
	w = httptest.NewRecorder()
	handler.ChangeRole(w, r)
	if w.Code != 303 {
		t.Fatalf("role status %d", w.Code)
	}
	changed, _ := users.FindByID(ctx, created.ID)
	if changed.Role != "general" {
		t.Error("role update did not persist")
	}
	if err := manager.ChangeRole(ctx, actor.ID, actor.ID, "demo"); !errors.Is(err, adminusers.ErrSelfDemotion) {
		t.Error("last administrator must be protected")
	}
	stillAdmin, _ := users.FindByID(ctx, actor.ID)
	if !stillAdmin.IsAdmin() {
		t.Error("administrator was demoted")
	}
}

func TestManagedRoleFilteringCombinesSearchPaginationAndLegacyRoles(t *testing.T) {
	database := regressionDatabase(t)
	ctx := context.Background()
	users := repository.NewUsersRepository(database)
	var docs []any
	for i := 0; i < 22; i++ {
		docs = append(docs, bson.M{"username": fmt.Sprintf("member-%02d", i), "role": "general", "passwordHash": "private-hash"})
	}
	docs = append(docs,
		bson.M{"username": "member-legacy-empty", "role": ""},
		bson.M{"username": "member-legacy-missing"},
		bson.M{"username": "member-legacy-null", "role": nil},
		bson.M{"username": "member-admin", "role": "admin"},
		bson.M{"username": "member-demo", "role": "demo"},
	)
	if _, err := database.Collection("users").InsertMany(ctx, docs); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		role           string
		total          int64
		page, wantRows int
	}{
		{"general", 25, 1, 20}, {"general", 25, 2, 5}, {"general", 25, 999, 5}, {"admin", 1, 1, 1}, {"demo", 1, 1, 1}, {"", 27, 1, 20},
	} {
		rows, count, err := users.ListManaged(ctx, "member", test.role, test.page, 20)
		if err != nil || count != test.total || len(rows) != test.wantRows {
			t.Fatalf("role=%q page=%d: count %d rows %d err %v", test.role, test.page, count, len(rows), err)
		}
		for _, row := range rows {
			if row.PasswordHash != "" {
				t.Error("filter listing leaked password hash")
			}
			if test.role != "" && row.EffectiveRole() != test.role {
				t.Error("role filter returned wrong account")
			}
		}
	}
	rows, count, err := users.ListManaged(ctx, "member-demo", "general", 1, 20)
	if err != nil || count != 0 || len(rows) != 0 {
		t.Error("name search and role filter must both apply")
	}
}
