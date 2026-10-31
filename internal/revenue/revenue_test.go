package revenue

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestCardRevenueService(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	storeID := uuid.New()

	t.Run("Card sale recording, fee calculation, and holiday shifting", func(t *testing.T) {
		// Thursday approval
		thursday := time.Date(2026, 10, 29, 14, 30, 0, 0, time.UTC) // Thursday
		payoutDate, shifted := svc.CalculatePayoutDate(thursday, false)

		// Thursday + 2 business days -> Friday(1), Monday(2) (Nov 2)
		if payoutDate.Weekday() != time.Monday {
			t.Errorf("expected payout to land on Monday, got %s", payoutDate.Weekday())
		}
		if !shifted {
			t.Errorf("expected holiday shifted to be true over weekend")
		}

		tx := &CardSaleTransaction{
			StoreID:          storeID,
			Acquirer:         AcquirerShinhan,
			ApprovalNumber:   "SH-882910",
			CardNumberMasked: "9411-****-****-1234",
			CardType:         "CREDIT",
			AuthorizedAmount: decimal.RequireFromString("100000"),
			ApprovedAt:       thursday,
		}

		err := svc.RecordCardSale(ctx, tx)
		if err != nil {
			t.Fatalf("failed to record card sale: %v", err)
		}

		// Verify 1.5% fee = 1,500 KRW, Net = 98,500 KRW
		expectedFee := decimal.RequireFromString("1500")
		expectedNet := decimal.RequireFromString("98500")

		if !tx.FeeAmount.Equal(expectedFee) {
			t.Errorf("expected fee %s, got %s", expectedFee, tx.FeeAmount)
		}
		if !tx.NetExpectedAmount.Equal(expectedNet) {
			t.Errorf("expected net expected %s, got %s", expectedNet, tx.NetExpectedAmount)
		}

		summary, err := svc.GetCardSalesSummary(ctx, storeID, time.Time{}, time.Time{}, nil)
		if err != nil {
			t.Fatalf("failed to get summary: %v", err)
		}

		if summary.TransactionCount != 1 {
			t.Errorf("expected 1 transaction, got %d", summary.TransactionCount)
		}
		if !summary.TotalAuthorized.Equal(decimal.RequireFromString("100000")) {
			t.Errorf("expected 100000 total authorized, got %s", summary.TotalAuthorized)
		}
		if !summary.TotalNetScheduled.Equal(expectedNet) {
			t.Errorf("expected %s net scheduled, got %s", expectedNet, summary.TotalNetScheduled)
		}
	})

	t.Run("Payout schedule creation and query", func(t *testing.T) {
		sched := &PayoutSchedule{
			StoreID:             storeID,
			PartnerCode:         "CARD_SHINHAN",
			ScheduledPayoutDate: time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC),
			GrossAmount:         decimal.RequireFromString("100000"),
			FeeAmount:           decimal.RequireFromString("1500"),
			OffsetAmount:        decimal.Zero,
			Status:              PayoutScheduled,
		}

		err := svc.CreatePayoutSchedule(ctx, sched)
		if err != nil {
			t.Fatalf("failed to create payout schedule: %v", err)
		}

		list, err := svc.ListPayoutSchedules(ctx, storeID, time.Time{}, time.Time{}, nil)
		if err != nil {
			t.Fatalf("failed to list payout schedules: %v", err)
		}
		if len(list) != 1 {
			t.Errorf("expected 1 schedule, got %d", len(list))
		}
		if !list[0].NetPayoutAmount.Equal(decimal.RequireFromString("98500")) {
			t.Errorf("expected net payout 98500, got %s", list[0].NetPayoutAmount)
		}
	})
}

