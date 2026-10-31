package pos

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"merchant-os/internal/ledger"
)

func TestPosService(t *testing.T) {
	ledgerSvc := ledger.NewService()
	svc := NewService(ledgerSvc)
	ctx := context.Background()
	storeID := uuid.New()

	t.Run("Ingest POS order paid and verify double-entry journal creation", func(t *testing.T) {
		event := &PosEvent{
			EventID:        "evt-pos-001",
			PosVendor:      "OKPOS",
			EventType:      EventOrderPaid,
			PosOrderNumber: "ORD-20261031-001",
			OrderTime:      time.Now(),
			TotalAmount:    decimal.RequireFromString("33000"),
			Payments: []PosPaymentMethod{
				{Method: "CARD", Amount: decimal.RequireFromString("33000"), ApprovalNumber: "77665544"},
			},
			Items: []PosOrderItem{
				{Name: "아메리카노", Quantity: 2, UnitPrice: decimal.RequireFromString("4500"), TotalPrice: decimal.RequireFromString("9000")},
				{Name: "파스타", Quantity: 2, UnitPrice: decimal.RequireFromString("12000"), TotalPrice: decimal.RequireFromString("24000")},
			},
		}

		entry, err := svc.IngestPosEvent(ctx, storeID, event)
		if err != nil {
			t.Fatalf("failed to ingest POS event: %v", err)
		}

		if entry == nil {
			t.Fatalf("expected journal entry to be created automatically")
		}

		// Check journal balance: Debit 33000 == Credit (30000 + 3000)
		if len(entry.Lines) != 3 {
			t.Errorf("expected 3 journal lines, got %d", len(entry.Lines))
		}
		if !entry.Lines[0].Debit.Equal(decimal.RequireFromString("33000")) {
			t.Errorf("expected debit 33000, got %s", entry.Lines[0].Debit)
		}
	})

	t.Run("Duplicate event ID is rejected with conflict", func(t *testing.T) {
		event := &PosEvent{
			EventID:        "evt-pos-001", // duplicate
			PosVendor:      "OKPOS",
			EventType:      EventOrderPaid,
			PosOrderNumber: "ORD-20261031-001",
			TotalAmount:    decimal.RequireFromString("33000"),
		}

		_, err := svc.IngestPosEvent(ctx, storeID, event)
		if err == nil {
			t.Fatalf("expected conflict on duplicate event_id")
		}
	})
}

