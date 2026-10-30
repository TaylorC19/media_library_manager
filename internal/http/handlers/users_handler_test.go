package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	domainauth "media_library_manager/internal/domain/auth"
	"media_library_manager/internal/http/middleware"
	"media_library_manager/internal/service/adminusers"
	"media_library_manager/internal/views"
)

type fakeManager struct {
	calls, creates int
	lastRole       string
	lastFilter     string
	rows           []domainauth.User
	err            error
}

func (f *fakeManager) List(_ context.Context, _ primitive.ObjectID, _ string, role string, _ int) ([]domainauth.User, int64, error) {
	f.calls++
	f.lastFilter = role
	return f.rows, int64(len(f.rows)), nil
}
func (f *fakeManager) ChangeRole(_ context.Context, _, _ primitive.ObjectID, role string) error {
	f.calls++
	f.lastRole = role
	return f.err
}
func (f *fakeManager) Create(_ context.Context, _ primitive.ObjectID, _, _, _, role string) error {
	f.creates++
	f.lastRole = role
	return f.err
}

func TestUserManagementHandlersRejectNonAdmins(t *testing.T) {
	for _, role := range []string{"", "general", "demo", "unknown"} {
		manager := &fakeManager{}
		handler := NewUsersHandler(nil, manager)
		for _, action := range []http.HandlerFunc{handler.Page, handler.Create, handler.ChangeRole} {
			r := httptest.NewRequest("POST", "/en/admin/users", nil)
			r = r.WithContext(middleware.WithUser(r.Context(), &domainauth.User{Role: role}))
			w := httptest.NewRecorder()
			action(w, r)
			if w.Code != 403 || manager.calls != 0 || manager.creates != 0 {
				t.Error("non-admin must not access account management")
			}
		}
	}
}

func TestUserManagementPageDoesNotLeakSecrets(t *testing.T) {
	renderer, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	actor := &domainauth.User{ID: primitive.NewObjectID(), Username: "admin", Role: "admin"}
	manager := &fakeManager{rows: []domainauth.User{*actor, {ID: primitive.NewObjectID(), Username: "demo", Role: "demo", PasswordHash: "secret-hash-never-show"}}}
	handler := NewUsersHandler(renderer, manager)
	r := httptest.NewRequest("GET", "/en/admin/users", nil)
	r = r.WithContext(middleware.WithUser(r.Context(), actor))
	w := httptest.NewRecorder()
	handler.Page(w, r)
	html := w.Body.String()
	if w.Code != 200 || !strings.Contains(html, "User management") || !strings.Contains(html, "Create account") {
		t.Fatal("page did not render")
	}
	if strings.Contains(html, "secret-hash-never-show") {
		t.Error("page leaked password hash")
	}
	if !strings.Contains(html, `name="role" disabled`) {
		t.Error("own role must be disabled")
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Error("account listing must not be cached")
	}
}

func TestRoleChangePreservesFiltersAndRejectsCrossOrigin(t *testing.T) {
	manager := &fakeManager{}
	handler := NewUsersHandler(nil, manager)
	id := primitive.NewObjectID()
	values := url.Values{"role": {"demo"}, "q": {"someone"}, "page": {"2"}, "filter_role": {"admin"}}
	r := httptest.NewRequest("POST", "/en/admin/users/"+id.Hex()+"/role", strings.NewReader(values.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	route := chi.NewRouteContext()
	route.URLParams.Add("userId", id.Hex())
	r = r.WithContext(context.WithValue(middleware.WithUser(r.Context(), &domainauth.User{ID: primitive.NewObjectID(), Role: "admin"}), chi.RouteCtxKey, route))
	w := httptest.NewRecorder()
	handler.ChangeRole(w, r)
	if w.Code != 303 || manager.lastRole != "demo" || (!strings.Contains(w.Header().Get("Location"), "q=someone") || !strings.Contains(w.Header().Get("Location"), "role=admin")) {
		t.Error("role change should preserve account search")
	}
	r.Header.Set("Origin", "https://elsewhere.invalid")
	w = httptest.NewRecorder()
	handler.ChangeRole(w, r)
	if w.Code != 403 || manager.calls != 1 {
		t.Error("cross-origin change should never reach manager")
	}
}

func TestCreateFailureDoesNotEchoPassword(t *testing.T) {
	renderer, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	manager := &fakeManager{err: adminusers.ErrInvalidRole}
	handler := NewUsersHandler(renderer, manager)
	r := httptest.NewRequest("POST", "/en/admin/users", strings.NewReader("username=newuser&password=secret-password-never-echo&role=unknown"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r = r.WithContext(middleware.WithUser(r.Context(), &domainauth.User{Role: "admin"}))
	w := httptest.NewRecorder()
	handler.Create(w, r)
	if strings.Contains(w.Body.String(), "secret-password-never-echo") {
		t.Error("creation errors must not echo passwords")
	}
	if !strings.Contains(w.Body.String(), `value="newuser"`) || !strings.Contains(w.Body.String(), `class="card users-create" open`) {
		t.Error("validation should preserve safe fields and keep creation open")
	}
}

func TestUserRoleFilterRendersAndIsPassedToManager(t *testing.T) {
	renderer, err := views.NewRenderer(true)
	if err != nil {
		t.Fatal(err)
	}
	manager := &fakeManager{}
	handler := NewUsersHandler(renderer, manager)
	r := httptest.NewRequest("GET", "/en/admin/users?q=member&role=demo", nil)
	r = r.WithContext(middleware.WithUser(r.Context(), &domainauth.User{Role: "admin"}))
	w := httptest.NewRecorder()
	handler.Page(w, r)
	if w.Code != 200 || manager.lastFilter != "demo" {
		t.Error("role filter was not passed to manager")
	}
	for _, want := range []string{`id="users-role" name="role"`, `value="demo" selected`, `name="filter_role" value="demo"`} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("missing %s", want)
		}
	}
}
