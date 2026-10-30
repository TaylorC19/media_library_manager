package adminusers

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
	domainauth "media_library_manager/internal/domain/auth"
)

type fakeUsers struct {
	mu    sync.Mutex
	users map[primitive.ObjectID]domainauth.User
	lists int
}

func (f *fakeUsers) FindByID(_ context.Context, id primitive.ObjectID) (*domainauth.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[id]
	if !ok {
		return nil, nil
	}
	return &u, nil
}
func (f *fakeUsers) ListManaged(_ context.Context, q, role string, _, _ int) ([]domainauth.User, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lists++
	var out []domainauth.User
	for _, u := range f.users {
		if strings.Contains(u.Username, q) && (role == "" || u.EffectiveRole() == role) {
			u.PasswordHash = ""
			out = append(out, u)
		}
	}
	return out, int64(len(out)), nil
}
func (f *fakeUsers) SetRole(_ context.Context, name, role string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for id, u := range f.users {
		if u.Username == name {
			u.Role = role
			f.users[id] = u
			return nil
		}
	}
	return ErrNotFound
}

type fakeCreator struct {
	calls int
	role  string
}

func (f *fakeCreator) CreateAccount(_ context.Context, _, _, _, role string) (*domainauth.User, error) {
	f.calls++
	f.role = role
	return &domainauth.User{Role: role}, nil
}

func TestAdminOnlyListingCreationAndRoleEditing(t *testing.T) {
	admin, general, target := primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID()
	users := &fakeUsers{users: map[primitive.ObjectID]domainauth.User{admin: {ID: admin, Username: "admin", Role: "admin"}, general: {ID: general, Username: "general", Role: "general"}, target: {ID: target, Username: "target", Role: "demo"}}}
	creator := &fakeCreator{}
	service := NewService(users, creator)
	ctx := context.Background()
	if _, _, err := service.List(ctx, general, "", "", 1); !errors.Is(err, ErrForbidden) || users.lists != 0 {
		t.Error("general user must not list accounts")
	}
	if err := service.Create(ctx, general, "new", "password", "", domainauth.RoleAdmin); !errors.Is(err, ErrForbidden) || creator.calls != 0 {
		t.Error("general user must not create admins")
	}
	if err := service.ChangeRole(ctx, general, target, "admin"); !errors.Is(err, ErrForbidden) {
		t.Error("general user must not change roles")
	}
	if err := service.ChangeRole(ctx, admin, target, "general"); err != nil {
		t.Fatal(err)
	}
	if err := service.ChangeRole(ctx, admin, admin, "demo"); !errors.Is(err, ErrSelfDemotion) {
		t.Error("admin must not demote self")
	}
	if err := service.Create(ctx, admin, "new", "password", "", domainauth.RoleDemo); err != nil || creator.calls != 1 || creator.role != "demo" {
		t.Error("admin should be able to create a demo account")
	}
	if err := service.ChangeRole(ctx, admin, target, "superadmin"); !errors.Is(err, ErrInvalidRole) {
		t.Error("unknown roles must be rejected")
	}
}

func TestSimultaneousCrossDemotionsLeaveAnAdministrator(t *testing.T) {
	a, b := primitive.NewObjectID(), primitive.NewObjectID()
	users := &fakeUsers{users: map[primitive.ObjectID]domainauth.User{a: {ID: a, Username: "a", Role: "admin"}, b: {ID: b, Username: "b", Role: "admin"}}}
	service := NewService(users, &fakeCreator{})
	start := make(chan struct{})
	results := make(chan error, 2)
	go func() { <-start; results <- service.ChangeRole(context.Background(), a, b, "general") }()
	go func() { <-start; results <- service.ChangeRole(context.Background(), b, a, "general") }()
	close(start)
	first, second := <-results, <-results
	if (first == nil) == (second == nil) {
		t.Error("exactly one demotion should succeed")
	}
	if first != nil && !errors.Is(first, ErrForbidden) {
		t.Fatal(first)
	}
	if second != nil && !errors.Is(second, ErrForbidden) {
		t.Fatal(second)
	}
	ua, _ := users.FindByID(context.Background(), a)
	ub, _ := users.FindByID(context.Background(), b)
	if !ua.IsAdmin() && !ub.IsAdmin() {
		t.Fatal("last admin lost")
	}
}

func TestUserListRoleFilterValidation(t *testing.T) {
	admin, legacy := primitive.NewObjectID(), primitive.NewObjectID()
	users := &fakeUsers{users: map[primitive.ObjectID]domainauth.User{admin: {ID: admin, Username: "admin", Role: "admin"}, legacy: {ID: legacy, Username: "legacy", Role: ""}}}
	service := NewService(users, &fakeCreator{})
	rows, total, err := service.List(context.Background(), admin, "legacy", "general", 1)
	if err != nil || len(rows) != 1 || total != 1 {
		t.Error("general role should include legacy accounts")
	}
	rows, total, err = service.List(context.Background(), admin, "legacy", "demo", 1)
	if err != nil || len(rows) != 0 || total != 0 {
		t.Error("role and query filters must be combined")
	}
	if _, _, err := service.List(context.Background(), admin, "", "invalid", 1); !errors.Is(err, ErrInvalidRole) {
		t.Error("unknown role filters must be rejected")
	}
}
