package auth

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
	domainauth "media_library_manager/internal/domain/auth"
	"media_library_manager/internal/repository"
)

// CreateAccount does not sign in as the new user. Callers must authorize any
// role other than general; public registration always supplies RoleGeneral.
func (s *Service) CreateAccount(ctx context.Context, username, password, displayName, role string) (*domainauth.User, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	displayName = strings.TrimSpace(displayName)
	if username == "" || len(username) > 80 || len(password) < 8 || len(password) > 72 || len(displayName) > 120 || !domainauth.IsRole(role) {
		return nil, ErrInvalidInput
	}
	for _, char := range username {
		if unicode.IsSpace(char) || unicode.IsControl(char) {
			return nil, ErrInvalidInput
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	if displayName == "" {
		displayName = username
	}
	user := &domainauth.User{Username: username, DisplayName: &displayName, PasswordHash: string(hash), Role: role}
	if err := s.usersRepo.Create(ctx, user); err != nil {
		if err == repository.ErrUsernameTaken {
			return nil, ErrUsernameTaken
		}
		return nil, err
	}
	return user, nil
}
