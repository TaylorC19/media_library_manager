// Package providerkeys stores personal provider credentials outside database
// backups and supplies immutable per-request overrides to provider clients.
package providerkeys

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var Providers = []string{"tmdb", "rawg", "discogs"}

type Keys map[string]string
type contextKey struct{}

func WithKeys(ctx context.Context, keys Keys) context.Context {
	snapshot := Keys{}
	for key, value := range keys {
		snapshot[key] = value
	}
	return context.WithValue(ctx, contextKey{}, snapshot)
}

func Resolve(ctx context.Context, provider, fallback string) string {
	if keys, ok := ctx.Value(contextKey{}).(Keys); ok {
		if value := strings.TrimSpace(keys[provider]); value != "" {
			return value
		}
	}
	return strings.TrimSpace(fallback)
}

type Store struct {
	dir string
	mu  sync.Mutex
}

func NewStore(dir string) *Store { return &Store{dir: dir} }

func (s *Store) Get(userID primitive.ObjectID) (Keys, error) {
	file, err := os.Open(filepath.Join(s.dir, userID.Hex()+".json"))
	if errors.Is(err, os.ErrNotExist) {
		return Keys{}, nil
	}
	if err != nil {
		return nil, errors.New("could not read provider keys")
	}
	defer file.Close()
	var keys Keys
	if err := json.NewDecoder(file).Decode(&keys); err != nil {
		return nil, errors.New("could not read provider keys")
	}
	return keys, nil
}

// Update changes only supplied fields; an empty value removes the personal
// override. Writes are atomic, with owner-only directory and file permissions.
func (s *Store) Update(userID primitive.ObjectID, changes Keys) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys, err := s.Get(userID)
	if err != nil {
		return err
	}
	if keys == nil {
		keys = Keys{}
	}
	for _, provider := range Providers {
		if value, ok := changes[provider]; ok {
			if value == "" {
				delete(keys, provider)
			} else {
				keys[provider] = value
			}
		}
	}
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return errors.New("could not save provider keys")
	}
	file, err := os.CreateTemp(s.dir, ".keys-*")
	if err != nil {
		return errors.New("could not save provider keys")
	}
	defer os.Remove(file.Name())
	if err := json.NewEncoder(file).Encode(keys); err != nil {
		file.Close()
		return errors.New("could not save provider keys")
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return errors.New("could not save provider keys")
	}
	if err := file.Close(); err != nil {
		return errors.New("could not save provider keys")
	}
	if err := os.Rename(file.Name(), filepath.Join(s.dir, userID.Hex()+".json")); err != nil {
		return errors.New("could not save provider keys")
	}
	return nil
}

// Status exposes only configuration flags, never the saved credential values.
func (s *Store) Status(userID primitive.ObjectID) (map[string]bool, error) {
	keys, err := s.Get(userID)
	if err != nil {
		return nil, err
	}
	status := map[string]bool{}
	for _, provider := range Providers {
		status[provider] = strings.TrimSpace(keys[provider]) != ""
	}
	return status, nil
}

// Clear removes only a personal override. Server defaults and other providers
// are untouched; Update uses the same lock/atomic write path as owner settings.
func (s *Store) Clear(userID primitive.ObjectID, provider string) error {
	for _, known := range Providers {
		if provider == known {
			return s.Update(userID, Keys{provider: ""})
		}
	}
	return errors.New("unsupported provider")
}
