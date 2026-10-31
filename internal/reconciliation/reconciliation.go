package reconciliation

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"merchant-os/internal/revenue"
)

// DepositCategory classifies incoming bank transactions
type DepositCategory string

const (
	CategoryCardSettlement     DepositCategory = "CARD_SETTLEMENT"
	CategoryDeliverySettlement DepositCategory = "DELIVERY_SETTLEMENT"
	CategoryPGSettlement       DepositCategory = "PG_SETTLEMENT"
	CategoryIgnoredNonRevenue  DepositCategory = "IGNORED_NON_REVENUE" // Owner personal funds, loans
	CategoryOther              DepositCategory = "OTHER"
)

// MatchType represents reconciliation matching resolution
type MatchType string

const (
	MatchExact1To1         MatchType = "EXACT_1_TO_1"
	MatchFeeTolerance      MatchType = "FEE_TOLERANCE_MATCH" // ±10 KRW fee rounding difference
	MatchHolidayShift      MatchType = "HOLIDAY_SHIFT"      // Shifted by 1~3 days
	MatchSplit1ToN         MatchType = "SPLIT_1_TO_N"
	MatchSyncDelayPending  MatchType = "SYNC_DELAY_PENDING" // External data lagging (not unpaid!)
	MatchDelayedCandidate  MatchType = "DELAYED_CANDIDATE"  // Suspicion of payout delay
	MatchConfirmedUnpaid   MatchType = "CONFIRMED_UNPAID"   // Unpaid after grace period
)

// BankDeposit represents an actual inbound deposit on merchant's account
type BankDeposit struct {
	ID               uuid.UUID       `json:"id"`
	StoreID          uuid.UUID       `json:"store_id"`
	BankCode         string          `json:"bank_code"`
	AccountNumber    string          `json:"account_number"`
	DepositedAt      time.Time       `json:"deposited_at"`
	Amount           decimal.Decimal `json:"amount"`
	CounterpartyMemo string          `json:"counterparty_memo"`
	PartnerCode      string          `json:"partner_code"`
	Category         DepositCategory `json:"category"`
	IsReconciled     bool            `json:"is_reconciled"`
}

// ReconciliationMatch represents single pairing between schedule and bank deposit
type ReconciliationMatch struct {
	ID             uuid.UUID       `json:"id"`
	ScheduleID     uuid.UUID       `json:"schedule_id"`
	DepositID      *uuid.UUID      `json:"deposit_id,omitempty"`
	PartnerCode    string          `json:"partner_code"`
	ScheduledDate  time.Time       `json:"scheduled_date"`
	ExpectedAmount decimal.Decimal `json:"expected_amount"`
	DepositedAmount decimal.Decimal `json:"deposited_amount"`
	VarianceAmount decimal.Decimal `json:"variance_amount"`
	MatchType      MatchType       `json:"match_type"`
	Confidence     float64         `json:"confidence"`
	Explanation    string          `json:"explanation"`
}

// ReconciliationReport summarizes the diagnostic run
type ReconciliationReport struct {
	StoreID             uuid.UUID              `json:"store_id"`
	AsOfDate            time.Time              `json:"as_of_date"`
	TotalScheduledCount int                    `json:"total_scheduled_count"`
	MatchedCount        int                    `json:"matched_count"`
	DelayedCount        int                    `json:"delayed_count"`
	SyncLagCount        int                    `json:"sync_lag_count"`
	UnpaidCount         int                    `json:"unpaid_count"`
	TotalMatchedAmount  decimal.Decimal        `json:"total_matched_amount"`
	TotalVarianceAmount decimal.Decimal        `json:"total_variance_amount"`
	Matches             []*ReconciliationMatch `json:"matches"`
}

// DisputeStatus represents dispute tracking state
type DisputeStatus string

const (
	DisputeOpen     DisputeStatus = "OPEN"
	DisputeResolved DisputeStatus = "RESOLVED"
	DisputeRejected DisputeStatus = "REJECTED"
)

// ReconciliationDispute represents a merchant's formal claim for unpaid amounts
type ReconciliationDispute struct {
	ID             uuid.UUID       `json:"id"`
	StoreID        uuid.UUID       `json:"store_id"`
	ScheduleID     uuid.UUID       `json:"schedule_id"`
	PartnerCode    string          `json:"partner_code"`
	DisputeType    string          `json:"dispute_type"` // UNPAID_DEFAULT, EXCESS_FEE, PARTIAL_DEPOSIT
	ClaimedAmount  decimal.Decimal `json:"claimed_amount"`
	EvidenceMemo   string          `json:"evidence_memo"`
	Status         DisputeStatus   `json:"status"`
	CreatedAt      time.Time       `json:"created_at"`
}

// Service manages bank deposits and automated reconciliation
type Service interface {
	RecordBankDeposit(ctx context.Context, dep *BankDeposit) error
	ListBankDeposits(ctx context.Context, storeID uuid.UUID, fromDate, toDate time.Time, category *DepositCategory) ([]*BankDeposit, error)
	RunDiagnostics(ctx context.Context, storeID uuid.UUID, asOfDate time.Time, schedules []*revenue.PayoutSchedule, lastBankSyncAt time.Time) (*ReconciliationReport, error)
	CreateDispute(ctx context.Context, dispute *ReconciliationDispute) error
	ListDisputes(ctx context.Context, storeID uuid.UUID) ([]*ReconciliationDispute, error)
	ParsePartnerFromMemo(memo string) (string, DepositCategory)
}

type serviceImpl struct {
	mu       sync.RWMutex
	deposits map[uuid.UUID][]*BankDeposit
	disputes map[uuid.UUID][]*ReconciliationDispute
}

func NewService() Service {
	return &serviceImpl{
		deposits: make(map[uuid.UUID][]*BankDeposit),
		disputes: make(map[uuid.UUID][]*ReconciliationDispute),
	}
}

// ParsePartnerFromMemo normalizes Korean bank memo into partner code and category
func (s *serviceImpl) ParsePartnerFromMemo(memo string) (string, DepositCategory) {
	upper := strings.ToUpper(memo)

	// Non-revenue detection first
	if strings.Contains(upper, "본인입금") || strings.Contains(upper, "사업주") || strings.Contains(upper, "대출") || strings.Contains(upper, "이자") {
		return "NON_REVENUE", CategoryIgnoredNonRevenue
	}

	// Card partners
	if strings.Contains(upper, "신한카드") || strings.Contains(upper, "SHINHAN") {
		return "CARD_SHINHAN", CategoryCardSettlement
	}
	if strings.Contains(upper, "국민카드") || strings.Contains(upper, "KB국민") {
		return "CARD_KB", CategoryCardSettlement
	}
	if strings.Contains(upper, "삼성카드") {
		return "CARD_SAMSUNG", CategoryCardSettlement
	}
	if strings.Contains(upper, "현대카드") {
		return "CARD_HYUNDAI", CategoryCardSettlement
	}
	if strings.Contains(upper, "롯데카드") {
		return "CARD_LOTTE", CategoryCardSettlement
	}
	if strings.Contains(upper, "우리카드") {
		return "CARD_WOORI", CategoryCardSettlement
	}
	if strings.Contains(upper, "하나카드") {
		return "CARD_HANA", CategoryCardSettlement
	}
	if strings.Contains(upper, "비씨카드") || strings.Contains(upper, "BC카드") {
		return "CARD_BC", CategoryCardSettlement
	}
	if strings.Contains(upper, "농협카드") || strings.Contains(upper, "NH카드") {
		return "CARD_NH", CategoryCardSettlement
	}

	// Delivery partners
	if strings.Contains(upper, "우아한") || strings.Contains(upper, "배민") {
		return "DELIVERY_BAEMIN", CategoryDeliverySettlement
	}
	if strings.Contains(upper, "쿠팡이츠") || strings.Contains(upper, "COUPANGEATS") {
		return "DELIVERY_COUPANG_EATS", CategoryDeliverySettlement
	}
	if strings.Contains(upper, "위대한상상") || strings.Contains(upper, "요기요") {
		return "DELIVERY_YOGIYO", CategoryDeliverySettlement
	}

	// PG partners
	if strings.Contains(upper, "토스페이") || strings.Contains(upper, "TOSS") {
		return "PG_TOSS", CategoryPGSettlement
	}
	if strings.Contains(upper, "나이스페이") {
		return "PG_NICE", CategoryPGSettlement
	}
	if strings.Contains(upper, "이니시스") || strings.Contains(upper, "INICIS") {
		return "PG_INICIS", CategoryPGSettlement
	}

	return "OTHER_INBOUND", CategoryOther
}

func (s *serviceImpl) RecordBankDeposit(ctx context.Context, dep *BankDeposit) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if dep.ID == uuid.Nil {
		dep.ID = uuid.New()
	}

	if dep.PartnerCode == "" || dep.Category == "" {
		code, cat := s.ParsePartnerFromMemo(dep.CounterpartyMemo)
		dep.PartnerCode = code
		dep.Category = cat
	}

	s.deposits[dep.StoreID] = append(s.deposits[dep.StoreID], dep)
	return nil
}

func (s *serviceImpl) ListBankDeposits(ctx context.Context, storeID uuid.UUID, fromDate, toDate time.Time, category *DepositCategory) ([]*BankDeposit, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deps := s.deposits[storeID]
	var result []*BankDeposit

	for _, dep := range deps {
		if !fromDate.IsZero() && dep.DepositedAt.Before(fromDate) {
			continue
		}
		if !toDate.IsZero() && dep.DepositedAt.After(toDate) {
			continue
		}
		if category != nil && dep.Category != *category {
			continue
		}
		result = append(result, dep)
	}

	return result, nil
}

// RunDiagnostics implements 4-way matching + honest sync lag vs unpaid distinction
func (s *serviceImpl) RunDiagnostics(ctx context.Context, storeID uuid.UUID, asOfDate time.Time, schedules []*revenue.PayoutSchedule, lastBankSyncAt time.Time) (*ReconciliationReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	deposits := s.deposits[storeID]

	report := &ReconciliationReport{
		StoreID:             storeID,
		AsOfDate:            asOfDate,
		TotalScheduledCount: len(schedules),
		TotalMatchedAmount:  decimal.Zero,
		TotalVarianceAmount: decimal.Zero,
	}

	usedDepositMap := make(map[uuid.UUID]bool)

	for _, sched := range schedules {
		matched := false

		// 1. Exact 1:1 Match
		for _, dep := range deposits {
			if usedDepositMap[dep.ID] || dep.Category == CategoryIgnoredNonRevenue {
				continue
			}

			if dep.PartnerCode == sched.PartnerCode && dep.Amount.Equal(sched.NetPayoutAmount) {
				// Date check within 1 day
				diffDays := dep.DepositedAt.Sub(sched.ScheduledPayoutDate).Hours() / 24
				if diffDays >= -1 && diffDays <= 1 {
					depID := dep.ID
					match := &ReconciliationMatch{
						ID:              uuid.New(),
						ScheduleID:      sched.ID,
						DepositID:       &depID,
						PartnerCode:     sched.PartnerCode,
						ScheduledDate:   sched.ScheduledPayoutDate,
						ExpectedAmount:  sched.NetPayoutAmount,
						DepositedAmount: dep.Amount,
						VarianceAmount:  decimal.Zero,
						MatchType:       MatchExact1To1,
						Confidence:      100.0,
						Explanation:     "정산예정액과 통장 입금액이 100% 일치합니다.",
					}
					report.Matches = append(report.Matches, match)
					report.MatchedCount++
					report.TotalMatchedAmount = report.TotalMatchedAmount.Add(dep.Amount)
					usedDepositMap[dep.ID] = true
					matched = true
					break
				}
			}
		}
		if matched {
			continue
		}

		// 2. Fee Tolerance Match (±10 KRW)
		for _, dep := range deposits {
			if usedDepositMap[dep.ID] || dep.Category == CategoryIgnoredNonRevenue {
				continue
			}

			if dep.PartnerCode == sched.PartnerCode {
				diff := dep.Amount.Sub(sched.NetPayoutAmount).Abs()
				if diff.LessThanOrEqual(decimal.NewFromInt(10)) && !diff.IsZero() {
					depID := dep.ID
					match := &ReconciliationMatch{
						ID:              uuid.New(),
						ScheduleID:      sched.ID,
						DepositID:       &depID,
						PartnerCode:     sched.PartnerCode,
						ScheduledDate:   sched.ScheduledPayoutDate,
						ExpectedAmount:  sched.NetPayoutAmount,
						DepositedAmount: dep.Amount,
						VarianceAmount:  dep.Amount.Sub(sched.NetPayoutAmount),
						MatchType:       MatchFeeTolerance,
						Confidence:      95.0,
						Explanation:     "카드사 원단위 절사/올림에 따른 미세 수수료 오차(±10원 이내)로 자동 대사되었습니다.",
					}
					report.Matches = append(report.Matches, match)
					report.MatchedCount++
					report.TotalMatchedAmount = report.TotalMatchedAmount.Add(dep.Amount)
					report.TotalVarianceAmount = report.TotalVarianceAmount.Add(match.VarianceAmount.Abs())
					usedDepositMap[dep.ID] = true
					matched = true
					break
				}
			}
		}
		if matched {
			continue
		}

		// 3. Holiday Shift Match (1~3 banking business days later)
		for _, dep := range deposits {
			if usedDepositMap[dep.ID] || dep.Category == CategoryIgnoredNonRevenue {
				continue
			}

			if dep.PartnerCode == sched.PartnerCode && dep.Amount.Equal(sched.NetPayoutAmount) {
				diffDays := dep.DepositedAt.Sub(sched.ScheduledPayoutDate).Hours() / 24
				if diffDays > 1 && diffDays <= 4 {
					depID := dep.ID
					match := &ReconciliationMatch{
						ID:              uuid.New(),
						ScheduleID:      sched.ID,
						DepositID:       &depID,
						PartnerCode:     sched.PartnerCode,
						ScheduledDate:   sched.ScheduledPayoutDate,
						ExpectedAmount:  sched.NetPayoutAmount,
						DepositedAmount: dep.Amount,
						VarianceAmount:  decimal.Zero,
						MatchType:       MatchHolidayShift,
						Confidence:      92.0,
						Explanation:     "은행/금융권 주말 및 공휴일로 인해 익영업일에 정상 입금되었습니다.",
					}
					report.Matches = append(report.Matches, match)
					report.MatchedCount++
					report.TotalMatchedAmount = report.TotalMatchedAmount.Add(dep.Amount)
					usedDepositMap[dep.ID] = true
					matched = true
					break
				}
			}
		}
		if matched {
			continue
		}

		// 4. Unmatched: Check if due date has passed
		if sched.ScheduledPayoutDate.Before(asOfDate) {
			// Sync lag check: If bank data timestamp was taken before payout date + 18h
			if !lastBankSyncAt.IsZero() && lastBankSyncAt.Before(sched.ScheduledPayoutDate.Add(18*time.Hour)) {
				match := &ReconciliationMatch{
					ID:              uuid.New(),
					ScheduleID:      sched.ID,
					PartnerCode:     sched.PartnerCode,
					ScheduledDate:   sched.ScheduledPayoutDate,
					ExpectedAmount:  sched.NetPayoutAmount,
					DepositedAmount: decimal.Zero,
					VarianceAmount:  sched.NetPayoutAmount,
					MatchType:       MatchSyncDelayPending,
					Confidence:      75.0,
					Explanation:     "은행 거래내역 동기화가 지연되고 있습니다. 미입금 사고가 아니며 최신 계좌 연동 후 자동 재대사됩니다.",
				}
				report.Matches = append(report.Matches, match)
				report.SyncLagCount++
			} else {
				// True candidate for delay or unpaid
				daysPast := asOfDate.Sub(sched.ScheduledPayoutDate).Hours() / 24
				matchType := MatchDelayedCandidate
				explanation := "지급예정일이 경과하였으나 입금 내역이 확인되지 않습니다 (입금 지연 후보)."
				if daysPast > 3 {
					matchType = MatchConfirmedUnpaid
					explanation = "지급예정일 3영업일 이상 경과 확정 미입금입니다. 이의제기를 등록하여 카드/플랫폼사에 소명을 요청하세요."
					report.UnpaidCount++
				} else {
					report.DelayedCount++
				}

				match := &ReconciliationMatch{
					ID:              uuid.New(),
					ScheduleID:      sched.ID,
					PartnerCode:     sched.PartnerCode,
					ScheduledDate:   sched.ScheduledPayoutDate,
					ExpectedAmount:  sched.NetPayoutAmount,
					DepositedAmount: decimal.Zero,
					VarianceAmount:  sched.NetPayoutAmount,
					MatchType:       matchType,
					Confidence:      88.0,
					Explanation:     explanation,
				}
				report.Matches = append(report.Matches, match)
				report.TotalVarianceAmount = report.TotalVarianceAmount.Add(sched.NetPayoutAmount)
			}
		}
	}

	return report, nil
}

func (s *serviceImpl) CreateDispute(ctx context.Context, dispute *ReconciliationDispute) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if dispute.ID == uuid.Nil {
		dispute.ID = uuid.New()
	}
	dispute.Status = DisputeOpen
	dispute.CreatedAt = time.Now().UTC()

	s.disputes[dispute.StoreID] = append(s.disputes[dispute.StoreID], dispute)
	return nil
}

func (s *serviceImpl) ListDisputes(ctx context.Context, storeID uuid.UUID) ([]*ReconciliationDispute, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.disputes[storeID], nil
}
