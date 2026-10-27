package identity

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"merchant-os/internal/platform/auth"
	apperr "merchant-os/internal/platform/errors"
)

// UserStatus represents user account lifecycle
type UserStatus string

const (
	StatusPendingVerification UserStatus = "PENDING_VERIFICATION"
	StatusActive              UserStatus = "ACTIVE"
	StatusSuspended           UserStatus = "SUSPENDED"
)

// User represents a platform user
type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Status       UserStatus `json:"status"`
	FullName     string     `json:"full_name"`
	MFAEnabled   bool       `json:"mfa_enabled"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Service manages user identity, authentication and sessions
type Service interface {
	Register(ctx context.Context, email, password, fullName string) (*User, error)
	Login(ctx context.Context, email, password string) (*User, string, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
}

type serviceImpl struct {
	mu    sync.RWMutex
	users map[string]*User
	byID  map[uuid.UUID]*User
}

func NewService() Service {
	return &serviceImpl{
		users: make(map[string]*User),
		byID:  make(map[uuid.UUID]*User),
	}
}

func (s *serviceImpl) Register(ctx context.Context, email, password, fullName string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if email == "" || password == "" {
		return nil, apperr.NewBadRequest("이메일과 비밀번호를 모두 입력해야 합니다.")
	}

	if _, exists := s.users[email]; exists {
		return nil, apperr.NewConflict("이미 등록된 이메일 주소입니다.")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperr.NewInternal(fmt.Errorf("failed to hash password: %w", err))
	}

	now := time.Now().UTC()
	user := &User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		Status:       StatusActive,
		FullName:     fullName,
		MFAEnabled:   false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	s.users[email] = user
	s.byID[user.ID] = user
	return user, nil
}

func (s *serviceImpl) Login(ctx context.Context, email, password string) (*User, string, error) {
	s.mu.RLock()
	user, exists := s.users[email]
	s.mu.RUnlock()

	if !exists {
		return nil, "", apperr.NewUnauthorized("이메일 또는 비밀번호가 일치하지 않습니다.")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", apperr.NewUnauthorized("이메일 또는 비밀번호가 일치하지 않습니다.")
	}

	token, err := auth.GenerateToken(user.ID, user.Email, "OWNER", uuid.Nil, uuid.Nil, 24*time.Hour)
	if err != nil {
		return nil, "", apperr.NewInternal(err)
	}

	return user, token, nil
}

func (s *serviceImpl) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.byID[id]
	if !exists {
		return nil, apperr.NewNotFound("사용자")
	}
	return user, nil
}
