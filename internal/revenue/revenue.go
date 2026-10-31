package revenue

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CardAcquirer represents Korean domestic card companies
type CardAcquirer string

const (
	AcquirerShinhan CardAcquirer = "SHINHAN"
	AcquirerKB      CardAcquirer = "KB"
	AcquirerSamsung CardAcquirer = "SAMSUNG"
	AcquirerHyundai CardAcquirer = "HYUNDAI"
	AcquirerLotte   CardAcquirer = "LOTTE"
	AcquirerWoori   CardAcquirer = "WOORI"
	AcquirerHana    CardAcquirer = "HANA"
	AcquirerBC      CardAcquirer = "BC"
	AcquirerNH      CardAcquirer = "NH"
)

// PayoutStatus represents settlement schedule status
type PayoutStatus string

const (
	PayoutScheduled        PayoutStatus = "SCHEDULED"
	PayoutDeposited        PayoutStatus = "DEPOSITED"
	PayoutOnHold           PayoutStatus = "ON_HOLD"
	PayoutDelayedCandidate PayoutStatus = "DELAYED_CANDIDATE"
)

// CardSaleTransaction represents single authorization/capture record
type CardSaleTransaction struct {
	ID                uuid.UUID       `json:"id"`
	StoreID           uuid.UUID       `json:"store_id"`
	Acquirer          CardAcquirer    `json:"acquirer"`
	ApprovalNumber    string          `json:"approval_number"`
	CardNumberMasked  string          `json:"card_number_masked"`
	CardType          string          `json:"card_type"` // CREDIT, CHECK_DEBIT
	AuthorizedAmount  decimal.Decimal `json:"authorized_amount"`
	CapturedAmount    decimal.Decimal `json:"captured_amount"`
	FeeAmount         decimal.Decimal `json:"fee_amount"`
	NetExpectedAmount decimal.Decimal `json:"net_expected_amount"`
	IsCanceled        bool            `json:"is_canceled"`
	ApprovedAt        time.Time       `json:"approved_at"`
	CapturedAt        *time.Time      `json:"captured_at,omitempty"`
	ScheduledPayoutAt time.Time       `json:"scheduled_payout_at"`
}

// CardSalesSummary represents aggregated card sales statistics
type CardSalesSummary struct {
	StoreID             uuid.UUID                  `json:"store_id"`
	TotalAuthorized     decimal.Decimal            `json:"total_authorized"`
	TotalCanceled       decimal.Decimal            `json:"total_canceled"`
	TotalCaptured       decimal.Decimal            `json:"total_captured"`
	TotalFee            decimal.Decimal            `json:"total_fee"`
	TotalNetScheduled   decimal.Decimal            `json:"total_net_scheduled"`
	TransactionCount    int                        `json:"transaction_count"`
	BreakdownByAcquirer map[CardAcquirer]AcquirerStat `json:"breakdown_by_acquirer"`
}

// AcquirerStat represents card sales per acquirer
type AcquirerStat struct {
	Count       int             `json:"count"`
	Authorized  decimal.Decimal `json:"authorized"`
	Fee         decimal.Decimal `json:"fee"`
	NetExpected decimal.Decimal `json:"net_expected"`
}

// PayoutSchedule represents a batch payout schedule
type PayoutSchedule struct {
	ID                  uuid.UUID       `json:"id"`
	StoreID             uuid.UUID       `json:"store_id"`
	PartnerCode         string          `json:"partner_code"` // CARD_SHINHAN, DELIVERY_BAEMIN etc.
	ScheduledPayoutDate time.Time       `json:"scheduled_payout_date"`
	GrossAmount         decimal.Decimal `json:"gross_amount"`
	FeeAmount           decimal.Decimal `json:"fee_amount"`
	OffsetAmount        decimal.Decimal `json:"offset_amount"`
	NetPayoutAmount     decimal.Decimal `json:"net_payout_amount"`
	HolidayShifted      bool            `json:"holiday_shifted"`
	Status              PayoutStatus    `json:"status"`
	CreatedAt           time.Time       `json:"created_at"`
}

// Service manages card sales and payout schedules
type Service interface {
	RecordCardSale(ctx context.Context, tx *CardSaleTransaction) error
	GetCardSalesSummary(ctx context.Context, storeID uuid.UUID, fromDate, toDate time.Time, acquirer *CardAcquirer) (*CardSalesSummary, error)
	CreatePayoutSchedule(ctx context.Context, sched *PayoutSchedule) error
	ListPayoutSchedules(ctx context.Context, storeID uuid.UUID, fromDate, toDate time.Time, status *PayoutStatus) ([]*PayoutSchedule, error)
	CalculatePayoutDate(approvedAt time.Time, isCheckCard bool) (time.Time, bool)
}

type serviceImpl struct {
	mu        sync.RWMutex
	sales     map[uuid.UUID][]*CardSaleTransaction
	schedules map[uuid.UUID][]*PayoutSchedule
}

func NewService() Service {
	return &serviceImpl{
		sales:     make(map[uuid.UUID][]*CardSaleTransaction),
		schedules: make(map[uuid.UUID][]*PayoutSchedule),
	}
}

// CalculatePayoutDate calculates D+2/D+3 banking business day with holiday shifting
func (s *serviceImpl) CalculatePayoutDate(approvedAt time.Time, isCheckCard bool) (time.Time, bool) {
	// Standard Korean Card Settlement: D+2 banking business days (Check card D+1 or D+2)
	daysToAdd := 2
	if isCheckCard {
		daysToAdd = 2
	}

	current := approvedAt
	added := 0
	holidayShifted := false

	for added < daysToAdd {
		current = current.AddDate(0, 0, 1)
		weekday := current.Weekday()
		if weekday != time.Saturday && weekday != time.Sunday {
			added++
		} else {
			holidayShifted = true
		}
	}

	// If landing on weekend, shift to Monday
	for current.Weekday() == time.Saturday || current.Weekday() == time.Sunday {
		current = current.AddDate(0, 0, 1)
		holidayShifted = true
	}

	// Normalise to 00:00:00 UTC
	return time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, time.UTC), holidayShifted
}

func (s *serviceImpl) RecordCardSale(ctx context.Context, tx *CardSaleTransaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if tx.ID == uuid.Nil {
		tx.ID = uuid.New()
	}

	// Calculate net expected = captured - fee
	if tx.CapturedAmount.IsZero() && !tx.IsCanceled {
		tx.CapturedAmount = tx.AuthorizedAmount
	}

	// Fee computation: 1.5% default if zero
	if tx.FeeAmount.IsZero() && !tx.AuthorizedAmount.IsZero() {
		tx.FeeAmount = tx.AuthorizedAmount.Mul(decimal.NewFromFloat(0.015)).Floor()
	}

	tx.NetExpectedAmount = tx.CapturedAmount.Sub(tx.FeeAmount)

	s.sales[tx.StoreID] = append(s.sales[tx.StoreID], tx)
	return nil
}

func (s *serviceImpl) GetCardSalesSummary(ctx context.Context, storeID uuid.UUID, fromDate, toDate time.Time, acquirer *CardAcquirer) (*CardSalesSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	txs := s.sales[storeID]

	summary := &CardSalesSummary{
		StoreID:             storeID,
		TotalAuthorized:     decimal.Zero,
		TotalCanceled:       decimal.Zero,
		TotalCaptured:       decimal.Zero,
		TotalFee:            decimal.Zero,
		TotalNetScheduled:   decimal.Zero,
		BreakdownByAcquirer: make(map[CardAcquirer]AcquirerStat),
	}

	for _, tx := range txs {
		if !fromDate.IsZero() && tx.ApprovedAt.Before(fromDate) {
			continue
		}
		if !toDate.IsZero() && tx.ApprovedAt.After(toDate) {
			continue
		}
		if acquirer != nil && tx.Acquirer != *acquirer {
			continue
		}

		summary.TransactionCount++
		summary.TotalAuthorized = summary.TotalAuthorized.Add(tx.AuthorizedAmount)
		if tx.IsCanceled {
			summary.TotalCanceled = summary.TotalCanceled.Add(tx.AuthorizedAmount)
		} else {
			summary.TotalCaptured = summary.TotalCaptured.Add(tx.CapturedAmount)
			summary.TotalFee = summary.TotalFee.Add(tx.FeeAmount)
			summary.TotalNetScheduled = summary.TotalNetScheduled.Add(tx.NetExpectedAmount)
		}

		stat := summary.BreakdownByAcquirer[tx.Acquirer]
		stat.Count++
		stat.Authorized = stat.Authorized.Add(tx.AuthorizedAmount)
		stat.Fee = stat.Fee.Add(tx.FeeAmount)
		stat.NetExpected = stat.NetExpected.Add(tx.NetExpectedAmount)
		summary.BreakdownByAcquirer[tx.Acquirer] = stat
	}

	return summary, nil
}

func (s *serviceImpl) CreatePayoutSchedule(ctx context.Context, sched *PayoutSchedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sched.ID == uuid.Nil {
		sched.ID = uuid.New()
	}
	sched.NetPayoutAmount = sched.GrossAmount.Sub(sched.FeeAmount).Sub(sched.OffsetAmount)
	sched.CreatedAt = time.Now().UTC()

	s.schedules[sched.StoreID] = append(s.schedules[sched.StoreID], sched)
	return nil
}

func (s *serviceImpl) ListPayoutSchedules(ctx context.Context, storeID uuid.UUID, fromDate, toDate time.Time, status *PayoutStatus) ([]*PayoutSchedule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	schedules := s.schedules[storeID]
	var result []*PayoutSchedule

	for _, sched := range schedules {
		if !fromDate.IsZero() && sched.ScheduledPayoutDate.Before(fromDate) {
			continue
		}
		if !toDate.IsZero() && sched.ScheduledPayoutDate.After(toDate) {
			continue
		}
		if status != nil && sched.Status != *status {
			continue
		}
		result = append(result, sched)
	}

	return result, nil
}
