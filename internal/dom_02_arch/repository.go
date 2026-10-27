package dom_02_arch

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// Repository manages persistence for DOM-02-ARCH tasks
type Repository interface {
	CheckFingerprintExists(ctx context.Context, storeID uuid.UUID, fingerprint string) (bool, error)
	SaveFingerprint(ctx context.Context, storeID uuid.UUID, fingerprint string) error
}

type memoryRepo struct {
	mu           sync.RWMutex
	fingerprints map[string]bool
}

// NewMemoryRepository returns an in-memory repository for development and invariant testing
func NewMemoryRepository() Repository {
	return &memoryRepo{
		fingerprints: make(map[string]bool),
	}
}

func (r *memoryRepo) CheckFingerprintExists(ctx context.Context, storeID uuid.UUID, fingerprint string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := storeID.String() + ":" + fingerprint
	return r.fingerprints[key], nil
}

func (r *memoryRepo) SaveFingerprint(ctx context.Context, storeID uuid.UUID, fingerprint string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := storeID.String() + ":" + fingerprint
	r.fingerprints[key] = true
	return nil
}
