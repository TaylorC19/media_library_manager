package adminusers

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
	domainauth "media_library_manager/internal/domain/auth"
)

type fakePersonalKeys struct {
	reads, clears int
	target        primitive.ObjectID
	provider      string
}

func (f *fakePersonalKeys) Status(primitive.ObjectID) (map[string]bool, error) {
	f.reads++
	return map[string]bool{"tmdb": true}, nil
}
func (f *fakePersonalKeys) Clear(id primitive.ObjectID, provider string) error {
	f.clears++
	f.target = id
	f.provider = provider
	return nil
}

func TestAdminKeyManagementAuthorizesActorAndReturnsNoSecrets(t *testing.T) {
	admin, target := primitive.NewObjectID(), primitive.NewObjectID()
	users := &fakeUsers{users: map[primitive.ObjectID]domainauth.User{admin: {ID: admin, Username: "admin", Role: "admin"}, target: {ID: target, Username: "target", Role: "general", PasswordHash: "private-password-hash"}}}
	store := &fakePersonalKeys{}
	service := NewKeyService(NewService(users, &fakeCreator{}), store)
	ctx := context.Background()
	if _, err := service.Describe(ctx, target, target); !errors.Is(err, ErrForbidden) || store.reads != 0 {
		t.Error("non-admin must not inspect keys")
	}
	if err := service.Clear(ctx, target, target, "tmdb"); !errors.Is(err, ErrForbidden) || store.clears != 0 {
		t.Error("non-admin must not clear keys")
	}
	overview, err := service.Describe(ctx, admin, target)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(overview)
	if strings.Contains(string(body), "private-password-hash") || len(overview.Providers) != 3 || !overview.Providers[0].Saved {
		t.Error("overview must contain status only")
	}
	if err := service.Clear(ctx, admin, target, "musicbrainz"); !errors.Is(err, ErrInvalidProvider) || store.clears != 0 {
		t.Error("providers without saved keys cannot be cleared")
	}
	if err := service.Clear(ctx, admin, target, "tmdb"); err != nil || store.target != target || store.provider != "tmdb" {
		t.Error("clear must target exactly the selected account/provider")
	}
	if err := users.SetRole(ctx, "admin", "general"); err != nil {
		t.Fatal(err)
	}
	if err := service.Clear(ctx, admin, target, "rawg"); !errors.Is(err, ErrForbidden) || store.clears != 1 {
		t.Error("demoted actor must be rejected before clearing")
	}
}
