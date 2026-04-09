package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	domainauth "media_library_manager/internal/domain/auth"
	"media_library_manager/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUsernameTaken      = errors.New("username already exists")
	ErrInvalidInput       = errors.New("invalid auth input")
)

type Service struct {
	usersRepo    *repository.UsersRepository
	sessionsRepo *repository.SessionsRepository
	sessionTTL   time.Duration
}

func NewService(usersRepo *repository.UsersRepository, sessionsRepo *repository.SessionsRepository, sessionTTL time.Duration) *Service {
	return &Service{
		usersRepo:    usersRepo,
		sessionsRepo: sessionsRepo,
		sessionTTL:   sessionTTL,
	}
}

func (s *Service) Register(ctx context.Context, username, password string) (*domainauth.User, string, error) {
	username = strings.TrimSpace(strings.ToLower(username))
	if username == "" || len(password) < 8 {
		return nil, "", ErrInvalidInput
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("hash password: %w", err)
	}

	user := &domainauth.User{
		Username:     username,
		PasswordHash: string(hash),
		DisplayName:  username,
	}
	if err := s.usersRepo.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrUsernameTaken) {
			return nil, "", ErrUsernameTaken
		}
		return nil, "", err
	}

	token, err := s.newSessionForUser(ctx, user.ID, "", "")
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

func (s *Service) Login(ctx context.Context, username, password, userAgent, ipAddress string) (*domainauth.User, string, error) {
	username = strings.TrimSpace(strings.ToLower(username))
	user, err := s.usersRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, "", err
	}
	if user == nil {
		return nil, "", ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	token, err := s.newSessionForUser(ctx, user.ID, userAgent, ipAddress)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

func (s *Service) ResolveSession(ctx context.Context, rawToken string) (*domainauth.User, *domainauth.Session, error) {
	if rawToken == "" {
		return nil, nil, nil
	}

	session, err := s.sessionsRepo.FindByTokenHash(ctx, hashToken(rawToken))
	if err != nil {
		return nil, nil, err
	}
	if session == nil {
		return nil, nil, nil
	}

	if err := s.sessionsRepo.Touch(ctx, session.ID); err != nil {
		return nil, nil, err
	}

	user, err := s.usersRepo.FindByID(ctx, session.UserID)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		_ = s.sessionsRepo.DeleteByID(ctx, session.ID)
		return nil, nil, nil
	}

	return user, session, nil
}

func (s *Service) LogoutByToken(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	return s.sessionsRepo.DeleteByTokenHash(ctx, hashToken(rawToken))
}

func (s *Service) newSessionForUser(ctx context.Context, userID primitive.ObjectID, userAgent, ipAddress string) (string, error) {
	rawToken, err := generateToken(32)
	if err != nil {
		return "", err
	}

	session := &domainauth.Session{
		UserID:    userID,
		TokenHash: hashToken(rawToken),
		ExpiresAt: time.Now().UTC().Add(s.sessionTTL),
		UserAgent: strings.TrimSpace(userAgent),
		IPAddress: strings.TrimSpace(ipAddress),
	}
	if err := s.sessionsRepo.Create(ctx, session); err != nil {
		return "", err
	}

	return rawToken, nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func generateToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
