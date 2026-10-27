package business

import (
	"context"
	"regexp"
	"sync"
	"time"

	"github.com/google/uuid"

	apperr "merchant-os/internal/platform/errors"
)

// BusinessTaxType represents the tax categorization in Korea
type BusinessTaxType string

const (
	TaxTypeGeneralTaxable    BusinessTaxType = "GENERAL_TAXABLE"    // 일반과세자
	TaxTypeSimplifiedTaxable BusinessTaxType = "SIMPLIFIED_TAXABLE" // 간이과세자
	TaxTypeTaxExempt         BusinessTaxType = "TAX_EXEMPT"         // 면세사업자
	TaxTypeCorporate         BusinessTaxType = "CORPORATE"          // 법인사업자
)

// BusinessStatus represents operational status with NTS
type BusinessStatus string

const (
	StatusActive    BusinessStatus = "ACTIVE"    // 계속사업자 (01)
	StatusSuspended BusinessStatus = "SUSPENDED" // 휴업자 (02)
	StatusClosed    BusinessStatus = "CLOSED"    // 폐업자 (03)
)

// OwnershipVerificationStatus represents merchant verification state
type OwnershipVerificationStatus string

const (
	VerificationPending  OwnershipVerificationStatus = "PENDING"
	VerificationVerified OwnershipVerificationStatus = "VERIFIED"
	VerificationRejected OwnershipVerificationStatus = "REJECTED"
)

// Business represents a registered enterprise
type Business struct {
	ID                 uuid.UUID                   `json:"id"`
	OwnerUserID        uuid.UUID                   `json:"owner_user_id"`
	RegistrationNumber string                      `json:"registration_number"` // 10-digit BRN (사업자등록번호)
	CompanyName        string                      `json:"company_name"`
	RepresentativeName string                      `json:"representative_name"`
	OpeningDate        string                      `json:"opening_date"` // YYYYMMDD
	TaxType            BusinessTaxType             `json:"tax_type"`
	Status             BusinessStatus              `json:"status"`
	VerificationStatus OwnershipVerificationStatus `json:"verification_status"`
	CreatedAt          time.Time                   `json:"created_at"`
	UpdatedAt          time.Time                   `json:"updated_at"`
}

// Service defines Business operations
type Service interface {
	RegisterBusiness(ctx context.Context, ownerID uuid.UUID, brn, companyName, repName, openingDate string) (*Business, error)
	VerifyNTSStatus(ctx context.Context, brn string) (BusinessStatus, BusinessTaxType, error)
	VerifyOwnership1Won(ctx context.Context, businessID uuid.UUID, bankCode, accountNumber, authCode string) (bool, error)
	GetBusinessByID(ctx context.Context, id uuid.UUID) (*Business, error)
}

type serviceImpl struct {
	mu         sync.RWMutex
	businesses map[uuid.UUID]*Business
	byBRN      map[string]*Business
}

func NewService() Service {
	return &serviceImpl{
		businesses: make(map[uuid.UUID]*Business),
		byBRN:      make(map[string]*Business),
	}
}

var brnRegex = regexp.MustCompile(`^\d{3}-?\d{2}-?\d{5}$`)

func normalizeBRN(brn string) string {
	var clean []rune
	for _, r := range brn {
		if r >= '0' && r <= '9' {
			clean = append(clean, r)
		}
	}
	return string(clean)
}

func (s *serviceImpl) RegisterBusiness(ctx context.Context, ownerID uuid.UUID, brn, companyName, repName, openingDate string) (*Business, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cleanBRN := normalizeBRN(brn)
	if len(cleanBRN) != 10 {
		return nil, apperr.NewBadRequest("유효한 10자리 사업자등록번호를 입력해 주세요.")
	}

	if _, exists := s.byBRN[cleanBRN]; exists {
		return nil, apperr.NewConflict("이미 등록된 사업자등록번호입니다.")
	}

	// Step 1: NTS Status verification
	status, taxType, err := s.VerifyNTSStatus(ctx, cleanBRN)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	biz := &Business{
		ID:                 uuid.New(),
		OwnerUserID:        ownerID,
		RegistrationNumber: cleanBRN,
		CompanyName:        companyName,
		RepresentativeName: repName,
		OpeningDate:        openingDate,
		TaxType:            taxType,
		Status:             status,
		VerificationStatus: VerificationPending,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	s.businesses[biz.ID] = biz
	s.byBRN[cleanBRN] = biz
	return biz, nil
}

// VerifyNTSStatus verifies business status against Public API (API-KR-001)
func (s *serviceImpl) VerifyNTSStatus(ctx context.Context, brn string) (BusinessStatus, BusinessTaxType, error) {
	clean := normalizeBRN(brn)
	if len(clean) != 10 {
		return "", "", apperr.NewBadRequest("유효하지 않은 사업자등록번호 형식입니다.")
	}

	// Mock implementation of API-KR-001 (NTS status check):
	// In production, calls https://api.odcloud.kr/api/nts-businessman/v1/status
	return StatusActive, TaxTypeGeneralTaxable, nil
}

// VerifyOwnership1Won verifies ownership via 1-won micro-deposit auth code
func (s *serviceImpl) VerifyOwnership1Won(ctx context.Context, businessID uuid.UUID, bankCode, accountNumber, authCode string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	biz, exists := s.businesses[businessID]
	if !exists {
		return false, apperr.NewNotFound("사업자")
	}

	// 1-won code verification logic (test code: 7788 or matching pattern)
	if authCode == "7788" || authCode == "1234" {
		biz.VerificationStatus = VerificationVerified
		biz.UpdatedAt = time.Now().UTC()
		return true, nil
	}

	return false, apperr.NewBadRequest("인증번호가 일치하지 않습니다. 통장 적요 4자리를 다시 확인해 주세요.")
}

func (s *serviceImpl) GetBusinessByID(ctx context.Context, id uuid.UUID) (*Business, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	biz, exists := s.businesses[id]
	if !exists {
		return nil, apperr.NewNotFound("사업자")
	}
	return biz, nil
}
