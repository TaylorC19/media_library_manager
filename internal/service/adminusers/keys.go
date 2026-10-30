package adminusers

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"media_library_manager/internal/providerkeys"
)

var ErrInvalidProvider = errors.New("invalid provider")

type personalKeys interface {
	Status(primitive.ObjectID) (map[string]bool, error)
	Clear(primitive.ObjectID, string) error
}
type SavedKey struct {
	ID, Name string
	Saved    bool
}
type KeyOverview struct {
	UserID    primitive.ObjectID
	Username  string
	Providers []SavedKey
}
type KeyService struct {
	admin *Service
	keys  personalKeys
}

func NewKeyService(admin *Service, keys personalKeys) *KeyService {
	return &KeyService{admin: admin, keys: keys}
}

func (s *KeyService) Describe(ctx context.Context, actor, target primitive.ObjectID) (*KeyOverview, error) {
	if err := s.admin.authorize(ctx, actor); err != nil {
		return nil, err
	}
	user, err := s.admin.users.FindByID(ctx, target)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}
	status, err := s.keys.Status(target)
	if err != nil {
		return nil, err
	}
	result := &KeyOverview{UserID: user.ID, Username: user.Username}
	names := map[string]string{"tmdb": "TMDB", "rawg": "RAWG", "discogs": "Discogs"}
	for _, id := range providerkeys.Providers {
		result.Providers = append(result.Providers, SavedKey{ID: id, Name: names[id], Saved: status[id]})
	}
	return result, nil
}
func (s *KeyService) Clear(ctx context.Context, actor, target primitive.ObjectID, provider string) error {
	roleChanges.Lock()
	defer roleChanges.Unlock()
	if err := s.admin.authorize(ctx, actor); err != nil {
		return err
	}
	known := false
	for _, id := range providerkeys.Providers {
		if id == provider {
			known = true
			break
		}
	}
	if !known {
		return ErrInvalidProvider
	}
	user, err := s.admin.users.FindByID(ctx, target)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrNotFound
	}
	return s.keys.Clear(target, provider)
}
