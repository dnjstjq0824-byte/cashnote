package store

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	apperr "merchant-os/internal/platform/errors"
)

// Role represents RBAC role in store
type Role string

const (
	RoleOwner      Role = "OWNER"
	RoleCoOwner    Role = "CO_OWNER"
	RoleManager    Role = "MANAGER"
	RoleAccountant Role = "ACCOUNTANT"
	RoleTaxAgent   Role = "TAX_AGENT"
	RoleStaff      Role = "STAFF"
)

// Store represents a physical or online branch/store
type Store struct {
	ID           uuid.UUID `json:"id"`
	BusinessID   uuid.UUID `json:"business_id"`
	StoreName    string    `json:"store_name"`
	StoreCode    string    `json:"store_code"`
	Category     string    `json:"category"`
	Address      string    `json:"address"`
	PostalCode   string    `json:"postal_code"`
	PhoneNumber  string    `json:"phone_number"`
	Currency     string    `json:"currency"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// StoreMember represents membership and authorization of a user to a store
type StoreMember struct {
	ID        uuid.UUID `json:"id"`
	StoreID   uuid.UUID `json:"store_id"`
	UserID    uuid.UUID `json:"user_id"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// Service manages store and multi-branch operations
type Service interface {
	CreateStore(ctx context.Context, businessID uuid.UUID, ownerUserID uuid.UUID, name, category, address string) (*Store, error)
	GetStoreByID(ctx context.Context, storeID uuid.UUID) (*Store, error)
	ListStoresByBusiness(ctx context.Context, businessID uuid.UUID) ([]*Store, error)
	AddMember(ctx context.Context, storeID, userID uuid.UUID, role Role) (*StoreMember, error)
	CheckPermission(ctx context.Context, storeID, userID uuid.UUID, requiredRole Role) (bool, error)
}

type serviceImpl struct {
	mu      sync.RWMutex
	stores  map[uuid.UUID]*Store
	members map[string]*StoreMember // key: storeID:userID
}

func NewService() Service {
	return &serviceImpl{
		stores:  make(map[uuid.UUID]*Store),
		members: make(map[string]*StoreMember),
	}
}

func (s *serviceImpl) CreateStore(ctx context.Context, businessID uuid.UUID, ownerUserID uuid.UUID, name, category, address string) (*Store, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		return nil, apperr.NewBadRequest("매장 이름을 입력해 주세요.")
	}

	now := time.Now().UTC()
	st := &Store{
		ID:          uuid.New(),
		BusinessID:  businessID,
		StoreName:   name,
		StoreCode:   "STR-" + uuid.New().String()[:8],
		Category:    category,
		Address:     address,
		Currency:    "KRW",
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.stores[st.ID] = st

	// Automatically grant OWNER role to creator
	key := st.ID.String() + ":" + ownerUserID.String()
	s.members[key] = &StoreMember{
		ID:        uuid.New(),
		StoreID:   st.ID,
		UserID:    ownerUserID,
		Role:      RoleOwner,
		CreatedAt: now,
	}

	return st, nil
}

func (s *serviceImpl) GetStoreByID(ctx context.Context, storeID uuid.UUID) (*Store, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	st, exists := s.stores[storeID]
	if !exists {
		return nil, apperr.NewNotFound("매장")
	}
	return st, nil
}

func (s *serviceImpl) ListStoresByBusiness(ctx context.Context, businessID uuid.UUID) ([]*Store, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Store
	for _, st := range s.stores {
		if st.BusinessID == businessID {
			result = append(result, st)
		}
	}
	return result, nil
}

func (s *serviceImpl) AddMember(ctx context.Context, storeID, userID uuid.UUID, role Role) (*StoreMember, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.stores[storeID]; !exists {
		return nil, apperr.NewNotFound("매장")
	}

	key := storeID.String() + ":" + userID.String()
	mem := &StoreMember{
		ID:        uuid.New(),
		StoreID:   storeID,
		UserID:    userID,
		Role:      role,
		CreatedAt: time.Now().UTC(),
	}
	s.members[key] = mem
	return mem, nil
}

func roleWeight(r Role) int {
	switch r {
	case RoleOwner:
		return 100
	case RoleCoOwner:
		return 90
	case RoleManager:
		return 70
	case RoleAccountant:
		return 60
	case RoleTaxAgent:
		return 50
	case RoleStaff:
		return 20
	default:
		return 0
	}
}

func (s *serviceImpl) CheckPermission(ctx context.Context, storeID, userID uuid.UUID, requiredRole Role) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := storeID.String() + ":" + userID.String()
	mem, exists := s.members[key]
	if !exists {
		return false, apperr.NewForbidden("해당 매장에 대한 접근 권한이 없습니다.")
	}

	if roleWeight(mem.Role) >= roleWeight(requiredRole) {
		return true, nil
	}

	return false, apperr.NewForbidden("이 작업을 수행할 수 있는 권한 등급이 부족합니다.")
}

