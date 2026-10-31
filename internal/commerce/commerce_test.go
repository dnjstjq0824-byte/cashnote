package commerce

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"merchant-os/internal/ledger"
)

func TestCommerceService(t *testing.T) {
	ledgerSvc := ledger.NewService()
	svc := NewService(ledgerSvc)
	ctx := context.Background()
	storeID := uuid.New()

	t.Run("Create and cancel subscription", func(t *testing.T) {
		sub, err := svc.CreateSubscription(ctx, storeID, TierPlus)
		if err != nil {
			t.Fatalf("failed to create subscription: %v", err)
		}

		if !sub.MonthlyPrice.Equal(decimal.RequireFromString("16500")) {
			t.Errorf("expected 16,500 KRW monthly price, got %s", sub.MonthlyPrice)
		}

		cancelled, err := svc.CancelSubscription(ctx, storeID)
		if err != nil {
			t.Fatalf("failed to cancel: %v", err)
		}
		if cancelled.Status != "CANCELLED" {
			t.Errorf("expected CANCELLED status, got %s", cancelled.Status)
		}
	})

	t.Run("Create B2B marketplace order and verify automatic COGS journal entry", func(t *testing.T) {
		items := []MarketplaceItem{
			{ItemName: "원두 1kg", Quantity: 5, UnitPrice: decimal.RequireFromString("20000")}, // 100,000
			{ItemName: "테이크아웃 컵 1000개", Quantity: 1, UnitPrice: decimal.RequireFromString("45000")}, // 45,000
		}

		order, entry, err := svc.CreateMarketplaceOrder(ctx, storeID, "BN 커피상사", items, "PAY_LATER")
		if err != nil {
			t.Fatalf("failed to create marketplace order: %v", err)
		}

		expectedTotal := decimal.RequireFromString("145000")
		if !order.TotalAmount.Equal(expectedTotal) {
			t.Errorf("expected total %s, got %s", expectedTotal, order.TotalAmount)
		}

		if entry == nil {
			t.Fatalf("expected double-entry journal entry to be created")
		}

		// Verify debit is COGS 5010 for 145,000 and credit is Accounts Payable 2010 for 145,000
		if !entry.Lines[0].Debit.Equal(expectedTotal) || entry.Lines[0].AccountCode != "5010" {
			t.Errorf("expected debit 5010 with amount %s", expectedTotal)
		}
		if !entry.Lines[1].Credit.Equal(expectedTotal) || entry.Lines[1].AccountCode != "2010" {
			t.Errorf("expected credit 2010 with amount %s", expectedTotal)
		}
	})
}
