package pos

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"merchant-os/internal/ledger"
	apperr "merchant-os/internal/platform/errors"
)

// PosEventType represents incoming POS transaction events
type PosEventType string

const (
	EventOrderPaid      PosEventType = "ORDER_PAID"
	EventOrderCancelled PosEventType = "ORDER_CANCELLED"
	EventShiftClosed    PosEventType = "SHIFT_CLOSED"
)

// PosPaymentMethod represents payment split
type PosPaymentMethod struct {
	Method         string          `json:"method"` // CARD, CASH, SIMPLE_PAY
	Amount         decimal.Decimal `json:"amount"`
	ApprovalNumber string          `json:"approval_number,omitempty"`
}

// PosOrderItem represents single line item in order
type PosOrderItem struct {
	Name       string          `json:"name"`
	Quantity   int             `json:"quantity"`
	UnitPrice  decimal.Decimal `json:"unit_price"`
	TotalPrice decimal.Decimal `json:"total_price"`
}

// PosEvent represents incoming webhook payload
type PosEvent struct {
	EventID        string             `json:"event_id"`
	StoreID        uuid.UUID          `json:"store_id"`
	PosVendor      string             `json:"pos_vendor"` // OKPOS, IMU, PAYHERE, TOSSPOS
	EventType      PosEventType       `json:"event_type"`
	PosOrderNumber string             `json:"pos_order_number"`
	OrderTime      time.Time          `json:"order_time"`
	Payments       []PosPaymentMethod `json:"payments"`
	TotalAmount    decimal.Decimal    `json:"total_amount"`
	SupplyAmount   decimal.Decimal    `json:"supply_amount"`
	VatAmount      decimal.Decimal    `json:"vat_amount"`
	Items          []PosOrderItem     `json:"items,omitempty"`
}

// Service manages POS events and automatic journal entry posting
type Service interface {
	IngestPosEvent(ctx context.Context, storeID uuid.UUID, event *PosEvent) (*ledger.JournalEntry, error)
	ListEvents(ctx context.Context, storeID uuid.UUID) ([]*PosEvent, error)
}

type serviceImpl struct {
	mu        sync.RWMutex
	seenEvent map[string]bool
	events    map[uuid.UUID][]*PosEvent
	ledgerSvc ledger.Service
}

func NewService(ledgerSvc ledger.Service) Service {
	return &serviceImpl{
		seenEvent: make(map[string]bool),
		events:    make(map[uuid.UUID][]*PosEvent),
		ledgerSvc: ledgerSvc,
	}
}

func (s *serviceImpl) IngestPosEvent(ctx context.Context, storeID uuid.UUID, event *PosEvent) (*ledger.JournalEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	event.StoreID = storeID

	// Idempotency check: duplicate event_id is rejected
	if s.seenEvent[event.EventID] {
		return nil, apperr.NewConflict(fmt.Sprintf("이미 수신된 POS 이벤트입니다 (Event ID: %s)", event.EventID))
	}

	if event.TotalAmount.IsNegative() {
		return nil, apperr.NewBadRequest("주문 금액은 0 이상이어야 합니다.")
	}

	if event.SupplyAmount.IsZero() && !event.TotalAmount.IsZero() {
		// Auto-split 10/11 supply and 1/11 VAT
		event.SupplyAmount = event.TotalAmount.Div(decimal.NewFromFloat(1.1)).Floor()
		event.VatAmount = event.TotalAmount.Sub(event.SupplyAmount)
	}

	s.seenEvent[event.EventID] = true
	s.events[storeID] = append(s.events[storeID], event)

	// Automatic double-entry posting into Core Ledger!
	if s.ledgerSvc != nil && event.EventType == EventOrderPaid && !event.TotalAmount.IsZero() {
		// Debit Receivable (1020) or Cash (1040)
		debitAccount := "1020" // Default Card Receivable
		if len(event.Payments) > 0 && event.Payments[0].Method == "CASH" {
			debitAccount = "1040" // Cash on hand
		}

		lines := []ledger.JournalLine{
			{
				AccountCode: debitAccount,
				Debit:       event.TotalAmount,
				Credit:      decimal.Zero,
				Description: fmt.Sprintf("[%s] POS 주문 결제 (%s)", event.PosVendor, event.PosOrderNumber),
			},
			{
				AccountCode: "4010", // Product Revenue
				Debit:       decimal.Zero,
				Credit:      event.SupplyAmount,
				Description: "상품 매출 공급가액",
			},
			{
				AccountCode: "2020", // Output VAT
				Debit:       decimal.Zero,
				Credit:      event.VatAmount,
				Description: "매출 부가가치세 예수금",
			},
		}

		entry, err := s.ledgerSvc.PostJournalEntry(ctx, storeID, event.OrderTime, fmt.Sprintf("POS 주문 자동기장 (%s)", event.PosOrderNumber), lines)
		if err != nil {
			return nil, err
		}
		return entry, nil
	}

	return nil, nil
}

func (s *serviceImpl) ListEvents(ctx context.Context, storeID uuid.UUID) ([]*PosEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.events[storeID], nil
}
