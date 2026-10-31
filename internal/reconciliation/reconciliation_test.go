package reconciliation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"merchant-os/internal/revenue"
)

func TestReconciliationEngine(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	storeID := uuid.New()

	t.Run("Memo parsing extracts correct partner and separates non-revenue", func(t *testing.T) {
		code, cat := svc.ParsePartnerFromMemo("신한카드정산")
		if code != "CARD_SHINHAN" || cat != CategoryCardSettlement {
			t.Errorf("expected CARD_SHINHAN, got %s, %s", code, cat)
		}

		code, cat = svc.ParsePartnerFromMemo("(주)우아한형제들")
		if code != "DELIVERY_BAEMIN" || cat != CategoryDeliverySettlement {
			t.Errorf("expected DELIVERY_BAEMIN, got %s, %s", code, cat)
		}

		code, cat = svc.ParsePartnerFromMemo("사업주본인입금")
		if cat != CategoryIgnoredNonRevenue {
			t.Errorf("expected IGNORED_NON_REVENUE for owner deposit, got %s", cat)
		}
	})

	t.Run("Level 4 Multi-Way Automated Reconciliation", func(t *testing.T) {
		payoutDate := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)

		// 1. Exact match schedule
		sched1 := &revenue.PayoutSchedule{
			ID:                  uuid.New(),
			StoreID:             storeID,
			PartnerCode:         "CARD_SHINHAN",
			ScheduledPayoutDate: payoutDate,
			NetPayoutAmount:     decimal.RequireFromString("98500"),
		}

		// 2. Fee tolerance schedule (schedule is 50,000, bank deposit is 49,995 - 5 won variance)
		sched2 := &revenue.PayoutSchedule{
			ID:                  uuid.New(),
			StoreID:             storeID,
			PartnerCode:         "CARD_KB",
			ScheduledPayoutDate: payoutDate,
			NetPayoutAmount:     decimal.RequireFromString("50000"),
		}

		// 3. Holiday shift schedule (schedule on Friday, deposit arrived Tuesday)
		sched3 := &revenue.PayoutSchedule{
			ID:                  uuid.New(),
			StoreID:             storeID,
			PartnerCode:         "DELIVERY_BAEMIN",
			ScheduledPayoutDate: payoutDate,
			NetPayoutAmount:     decimal.RequireFromString("300000"),
		}

		// 4. Sync delayed schedule (bank sync lagging)
		sched4 := &revenue.PayoutSchedule{
			ID:                  uuid.New(),
			StoreID:             storeID,
			PartnerCode:         "CARD_SAMSUNG",
			ScheduledPayoutDate: payoutDate,
			NetPayoutAmount:     decimal.RequireFromString("150000"),
		}

		// 5. True unpaid candidate
		sched5 := &revenue.PayoutSchedule{
			ID:                  uuid.New(),
			StoreID:             storeID,
			PartnerCode:         "CARD_HYUNDAI",
			ScheduledPayoutDate: payoutDate,
			NetPayoutAmount:     decimal.RequireFromString("70000"),
		}

		// Bank Deposits
		// Match 1
		_ = svc.RecordBankDeposit(ctx, &BankDeposit{
			StoreID:          storeID,
			BankCode:         "004",
			AccountNumber:    "110-123",
			DepositedAt:      payoutDate,
			Amount:           decimal.RequireFromString("98500"),
			CounterpartyMemo: "신한카드",
		})

		// Match 2 (Fee tolerance)
		_ = svc.RecordBankDeposit(ctx, &BankDeposit{
			StoreID:          storeID,
			BankCode:         "004",
			AccountNumber:    "110-123",
			DepositedAt:      payoutDate,
			Amount:           decimal.RequireFromString("49995"), // -5 KRW diff
			CounterpartyMemo: "국민카드",
		})

		// Match 3 (Holiday shift: +2 days)
		_ = svc.RecordBankDeposit(ctx, &BankDeposit{
			StoreID:          storeID,
			BankCode:         "004",
			AccountNumber:    "110-123",
			DepositedAt:      payoutDate.AddDate(0, 0, 2),
			Amount:           decimal.RequireFromString("300000"),
			CounterpartyMemo: "우아한형제들",
		})

		asOfDate := payoutDate.AddDate(0, 0, 4) // 4 days later
		// Last bank sync is fresh (current)
		lastBankSyncAt := asOfDate

		report, err := svc.RunDiagnostics(ctx, storeID, asOfDate, []*revenue.PayoutSchedule{sched1, sched2, sched3, sched5}, lastBankSyncAt)
		if err != nil {
			t.Fatalf("failed to run diagnostics: %v", err)
		}

		if report.MatchedCount != 3 {
			t.Errorf("expected 3 matched items, got %d", report.MatchedCount)
		}
		if report.UnpaidCount+report.DelayedCount != 1 {
			t.Errorf("expected 1 delayed/unpaid item for Hyundai, got delayed=%d, unpaid=%d", report.DelayedCount, report.UnpaidCount)
		}

		// Test Sync Delay: If bank sync is old (before payoutDate + 18h)
		oldBankSync := payoutDate.Add(10 * time.Hour)
		reportSyncLag, err := svc.RunDiagnostics(ctx, storeID, asOfDate, []*revenue.PayoutSchedule{sched4}, oldBankSync)
		if err != nil {
			t.Fatalf("failed to run sync lag diagnostics: %v", err)
		}

		if reportSyncLag.SyncLagCount != 1 {
			t.Errorf("expected 1 sync lag item, got %d", reportSyncLag.SyncLagCount)
		}
		if reportSyncLag.UnpaidCount != 0 {
			t.Errorf("sync lag must NOT be counted as unpaid default!")
		}
	})

	t.Run("Dispute creation and listing", func(t *testing.T) {
		dispute := &ReconciliationDispute{
			StoreID:       storeID,
			ScheduleID:    uuid.New(),
			PartnerCode:   "CARD_HYUNDAI",
			DisputeType:   "UNPAID_DEFAULT",
			ClaimedAmount: decimal.RequireFromString("70000"),
			EvidenceMemo:  "지급일 이후 미입금 확인 요청",
		}

		err := svc.CreateDispute(ctx, dispute)
		if err != nil {
			t.Fatalf("failed to create dispute: %v", err)
		}

		disputes, err := svc.ListDisputes(ctx, storeID)
		if err != nil || len(disputes) != 1 {
			t.Fatalf("expected 1 dispute, got %d", len(disputes))
		}
		if disputes[0].Status != DisputeOpen {
			t.Errorf("expected dispute status OPEN, got %s", disputes[0].Status)
		}
	})
}

