package adminusers

import (
	"context"
	"errors"
	"strings"
	"sync"

	"go.mongodb.org/mongo-driver/bson/primitive"

	domainauth "media_library_manager/internal/domain/auth"
)

var (
	ErrForbidden    = errors.New("administrator access required")
	ErrSelfDemotion = errors.New("cannot remove your own administrator access")
	ErrNotFound     = errors.New("account not found")
	ErrInvalidRole  = errors.New("invalid role")
)

type userRepository interface {
	FindByID(context.Context, primitive.ObjectID) (*domainauth.User, error)
	ListManaged(context.Context, string, string, int, int) ([]domainauth.User, int64, error)
	SetRole(context.Context, string, string) error
}
type accountCreator interface {
	CreateAccount(context.Context, string, string, string, string) (*domainauth.User, error)
}
type Service struct {
	users    userRepository
	accounts accountCreator
}

// The mutex supports in-memory repositories in unit tests. Mongo repositories
// also serialize authorization and updates in a shared database transaction.
var roleChanges sync.Mutex

func NewService(users userRepository, accounts accountCreator) *Service {
	return &Service{users: users, accounts: accounts}
}
func (s *Service) authorize(ctx context.Context, actor primitive.ObjectID) error {
	user, err := s.users.FindByID(ctx, actor)
	if err != nil {
		return err
	}
	if !user.IsAdmin() {
		return ErrForbidden
	}
	return nil
}
func (s *Service) List(ctx context.Context, actor primitive.ObjectID, query, role string, page int) ([]domainauth.User, int64, error) {
	if err := s.authorize(ctx, actor); err != nil {
		return nil, 0, err
	}
	if role != "" && !domainauth.IsRole(role) {
		return nil, 0, ErrInvalidRole
	}
	return s.users.ListManaged(ctx, strings.TrimSpace(query), role, page, 20)
}
func (s *Service) ChangeRole(ctx context.Context, actor, target primitive.ObjectID, role string) error {
	if coordinator, ok := s.users.(interface {
		WithRoleChange(context.Context, func(context.Context) error) error
	}); ok {
		return coordinator.WithRoleChange(ctx, func(ctx context.Context) error { return s.changeRole(ctx, actor, target, role) })
	}
	roleChanges.Lock()
	defer roleChanges.Unlock()
	return s.changeRole(ctx, actor, target, role)
}
func (s *Service) changeRole(ctx context.Context, actor, target primitive.ObjectID, role string) error {
	if !domainauth.IsRole(role) {
		return ErrInvalidRole
	}
	if err := s.authorize(ctx, actor); err != nil {
		return err
	}
	if actor == target && role != domainauth.RoleAdmin {
		return ErrSelfDemotion
	}
	user, err := s.users.FindByID(ctx, target)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrNotFound
	}
	return s.users.SetRole(ctx, user.Username, role)
}
func (s *Service) Create(ctx context.Context, actor primitive.ObjectID, username, password, displayName, role string) error {
	if coordinator, ok := s.users.(interface {
		WithRoleChange(context.Context, func(context.Context) error) error
	}); ok {
		return coordinator.WithRoleChange(ctx, func(ctx context.Context) error { return s.create(ctx, actor, username, password, displayName, role) })
	}
	roleChanges.Lock()
	defer roleChanges.Unlock()
	return s.create(ctx, actor, username, password, displayName, role)
}
func (s *Service) create(ctx context.Context, actor primitive.ObjectID, username, password, displayName, role string) error {
	if err := s.authorize(ctx, actor); err != nil {
		return err
	}
	if !domainauth.IsRole(role) {
		return ErrInvalidRole
	}
	_, err := s.accounts.CreateAccount(ctx, username, password, displayName, role)
	return err
}
