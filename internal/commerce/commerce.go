package commerce

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

// SubscriptionTier represents SaaS membership levels
type SubscriptionTier string

const (
	TierBasic SubscriptionTier = "BASIC" // Free starter
	TierPlus  SubscriptionTier = "PLUS"  // 16,500 KRW / mo
	TierPro   SubscriptionTier = "PRO"   // 33,000 KRW / mo
)

// Subscription represents store membership subscription
type Subscription struct {
	ID           uuid.UUID        `json:"id"`
	StoreID      uuid.UUID        `json:"store_id"`
	Tier         SubscriptionTier `json:"tier"`
	MonthlyPrice decimal.Decimal  `json:"monthly_price"`
	Status       string           `json:"status"` // ACTIVE, CANCELLED
	StartedAt    time.Time        `json:"started_at"`
	ExpiresAt    time.Time        `json:"expires_at"`
}

// MarketplaceItem represents an item ordered from restaurant supply catalog
type MarketplaceItem struct {
	ItemName  string          `json:"item_name"`
	Quantity  int             `json:"quantity"`
	UnitPrice decimal.Decimal `json:"unit_price"`
	Total     decimal.Decimal `json:"total"`
}

// MarketplaceOrder represents B2B supply/ingredients purchase
type MarketplaceOrder struct {
	ID            uuid.UUID         `json:"id"`
	StoreID       uuid.UUID         `json:"store_id"`
	SupplierName  string            `json:"supplier_name"`
	Items         []MarketplaceItem `json:"items"`
	TotalAmount   decimal.Decimal   `json:"total_amount"`
	PaymentMethod string            `json:"payment_method"` // PREPAID, PAY_LATER
	Status        string            `json:"status"`         // COMPLETED, DELIVERED
	CreatedAt     time.Time         `json:"created_at"`
}

// Service manages subscriptions and B2B marketplace
type Service interface {
	CreateSubscription(ctx context.Context, storeID uuid.UUID, tier SubscriptionTier) (*Subscription, error)
	CancelSubscription(ctx context.Context, storeID uuid.UUID) (*Subscription, error)
	GetSubscription(ctx context.Context, storeID uuid.UUID) (*Subscription, error)
	CreateMarketplaceOrder(ctx context.Context, storeID uuid.UUID, supplier string, items []MarketplaceItem, paymentMethod string) (*MarketplaceOrder, *ledger.JournalEntry, error)
}

type serviceImpl struct {
	mu            sync.RWMutex
	subscriptions map[uuid.UUID]*Subscription
	orders        map[uuid.UUID][]*MarketplaceOrder
	ledgerSvc     ledger.Service
}

func NewService(ledgerSvc ledger.Service) Service {
	return &serviceImpl{
		subscriptions: make(map[uuid.UUID]*Subscription),
		orders:        make(map[uuid.UUID][]*MarketplaceOrder),
		ledgerSvc:     ledgerSvc,
	}
}

func (s *serviceImpl) CreateSubscription(ctx context.Context, storeID uuid.UUID, tier SubscriptionTier) (*Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	price := decimal.Zero
	switch tier {
	case TierPlus:
		price = decimal.RequireFromString("16500")
	case TierPro:
		price = decimal.RequireFromString("33000")
	case TierBasic:
		price = decimal.Zero
	default:
		return nil, apperr.NewBadRequest("유효하지 않은 멤버십 플랜입니다.")
	}

	now := time.Now().UTC()
	sub := &Subscription{
		ID:           uuid.New(),
		StoreID:      storeID,
		Tier:         tier,
		MonthlyPrice: price,
		Status:       "ACTIVE",
		StartedAt:    now,
		ExpiresAt:    now.AddDate(0, 1, 0),
	}

	s.subscriptions[storeID] = sub
	return sub, nil
}

func (s *serviceImpl) CancelSubscription(ctx context.Context, storeID uuid.UUID) (*Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub, exists := s.subscriptions[storeID]
	if !exists {
		return nil, apperr.NewNotFound("구독 정보")
	}

	sub.Status = "CANCELLED"
	return sub, nil
}

func (s *serviceImpl) GetSubscription(ctx context.Context, storeID uuid.UUID) (*Subscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sub, exists := s.subscriptions[storeID]
	if !exists {
		// Default to Basic
		return &Subscription{
			StoreID:      storeID,
			Tier:         TierBasic,
			MonthlyPrice: decimal.Zero,
			Status:       "ACTIVE",
		}, nil
	}
	return sub, nil
}

// CreateMarketplaceOrder places order and automatically journals COGS & Accounts Payable into Ledger
func (s *serviceImpl) CreateMarketplaceOrder(ctx context.Context, storeID uuid.UUID, supplier string, items []MarketplaceItem, paymentMethod string) (*MarketplaceOrder, *ledger.JournalEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(items) == 0 {
		return nil, nil, apperr.NewBadRequest("주문할 상품 항목을 최소 1개 이상 입력하세요.")
	}

	total := decimal.Zero
	for i := range items {
		items[i].Total = items[i].UnitPrice.Mul(decimal.NewFromInt(int64(items[i].Quantity)))
		total = total.Add(items[i].Total)
	}

	now := time.Now().UTC()
	order := &MarketplaceOrder{
		ID:            uuid.New(),
		StoreID:       storeID,
		SupplierName:  supplier,
		Items:         items,
		TotalAmount:   total,
		PaymentMethod: paymentMethod,
		Status:        "COMPLETED",
		CreatedAt:     now,
	}

	s.orders[storeID] = append(s.orders[storeID], order)

	var entry *ledger.JournalEntry
	if s.ledgerSvc != nil && !total.IsZero() {
		// Credit account: 1010 (Bank deposit) if prepaid, or 2010 (Accounts Payable) if pay later
		creditAccount := "2010" // 외상매입금
		creditDesc := fmt.Sprintf("[%s] B2B 식자재 외상 매입", supplier)
		if paymentMethod == "PREPAID" {
			creditAccount = "1010" // 보통예금
			creditDesc = fmt.Sprintf("[%s] B2B 식자재 즉시 결제", supplier)
		}

		lines := []ledger.JournalLine{
			{
				AccountCode: "5010", // 식자재/원재료 매입(매출원가)
				Debit:       total,
				Credit:      decimal.Zero,
				Description: fmt.Sprintf("[%s] 식자재/소모품 매입", supplier),
			},
			{
				AccountCode: creditAccount,
				Debit:       decimal.Zero,
				Credit:      total,
				Description: creditDesc,
			},
		}

		var err error
		entry, err = s.ledgerSvc.PostJournalEntry(ctx, storeID, now, fmt.Sprintf("B2B 마켓 주문 자동 분개 (%s)", order.ID.String()[:8]), lines)
		if err != nil {
			return nil, nil, err
		}
	}

	return order, entry, nil
}
