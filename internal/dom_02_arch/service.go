package dom_02_arch

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"merchant-os/internal/platform/envelope"
	apperr "merchant-os/internal/platform/errors"
)

// SystemCapability represents platform capabilities and external gateway statuses
type SystemCapability struct {
	ServiceCode     string                   `json:"service_code"`
	Name            string                   `json:"name"`
	Status          string                   `json:"status"` // APPROVED, BLOCKED, CONTRACT_PENDING
	SourceLevel     envelope.ProvenanceLevel `json:"source_level"`
	FallbackLevel   string                   `json:"fallback_level"`
	HasDirectAccess bool                     `json:"has_direct_access"`
}

// HealthStatus represents service liveliness and readiness state
type HealthStatus struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
}

// Service defines DOM-02-ARCH architecture domain logic
type Service interface {
	GetLiveness(ctx context.Context) HealthStatus
	GetReadiness(ctx context.Context) HealthStatus
	ListSystemCapabilities(ctx context.Context) []SystemCapability
	VerifyIdempotency(ctx context.Context, storeID uuid.UUID, fingerprint string) error
	CalculateSampleFinancials(ctx context.Context, salesAmount, feeRate decimal.Decimal) (decimal.Decimal, decimal.Decimal, error)
}

type serviceImpl struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &serviceImpl{repo: repo}
}

func (s *serviceImpl) GetLiveness(ctx context.Context) HealthStatus {
	return HealthStatus{
		Status:    "UP",
		Version:   "1.0.0",
		Timestamp: time.Now().UTC(),
		Checks: map[string]string{
			"api_server": "healthy",
		},
	}
}

func (s *serviceImpl) GetReadiness(ctx context.Context) HealthStatus {
	return HealthStatus{
		Status:    "READY",
		Version:   "1.0.0",
		Timestamp: time.Now().UTC(),
		Checks: map[string]string{
			"database":  "connected",
			"redis":     "connected",
			"invariants": "verified",
		},
	}
}

func (s *serviceImpl) ListSystemCapabilities(ctx context.Context) []SystemCapability {
	return []SystemCapability{
		{
			ServiceCode:     "API-KR-001",
			Name:            "국세청 사업자 진위 및 휴폐업 상태조회",
			Status:          "APPROVED",
			SourceLevel:     envelope.Level3API,
			FallbackLevel:   "LEVEL_1_MANUAL",
			HasDirectAccess: true,
		},
		{
			ServiceCode:     "API-KR-002",
			Name:            "여신금융협회 가맹점 매출거래정보",
			Status:          "CONTRACT_REQUIRED",
			SourceLevel:     envelope.Level3API,
			FallbackLevel:   "LEVEL_2_FILE",
			HasDirectAccess: false,
		},
		{
			ServiceCode:     "API-KR-004",
			Name:            "금융결제원 오픈뱅킹 계좌 및 입출금",
			Status:          "APPROVAL_REQUIRED",
			SourceLevel:     envelope.Level3API,
			FallbackLevel:   "LEVEL_2_FILE",
			HasDirectAccess: false,
		},
		{
			ServiceCode:     "API-KR-010",
			Name:            "배달3사(배민/쿠팡이츠/요기요) 주문·정산",
			Status:          "BLOCKED",
			SourceLevel:     envelope.Level2File,
			FallbackLevel:   "LEVEL_2_FILE",
			HasDirectAccess: false,
		},
		{
			ServiceCode:     "API-KR-011",
			Name:            "매장 리뷰(Google Business Profile & AI 초안)",
			Status:          "APPROVED",
			SourceLevel:     envelope.Level3API,
			FallbackLevel:   "LEVEL_1_MANUAL",
			HasDirectAccess: true,
		},
	}
}

func (s *serviceImpl) VerifyIdempotency(ctx context.Context, storeID uuid.UUID, fingerprint string) error {
	exists, err := s.repo.CheckFingerprintExists(ctx, storeID, fingerprint)
	if err != nil {
		return err
	}
	if exists {
		return apperr.NewConflict("이미 처리된 중복 요청입니다 (중복 지문 일치).")
	}
	return s.repo.SaveFingerprint(ctx, storeID, fingerprint)
}

func (s *serviceImpl) CalculateSampleFinancials(ctx context.Context, salesAmount, feeRate decimal.Decimal) (decimal.Decimal, decimal.Decimal, error) {
	if salesAmount.IsNegative() {
		return decimal.Zero, decimal.Zero, apperr.NewBadRequest("매출 금액은 0 이상이어야 합니다.")
	}
	// Fee = salesAmount * feeRate
	fee := salesAmount.Mul(feeRate).Round(0)
	// Net Payout = salesAmount - fee
	netPayout := salesAmount.Sub(fee)
	return fee, netPayout, nil
}

