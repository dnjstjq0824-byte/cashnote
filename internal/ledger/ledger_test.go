package ledger

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestLedgerEngine(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	storeID := uuid.New()
	now := time.Now()

	t.Run("Card sale double-entry posting and P&L calculation", func(t *testing.T) {
		// Sales transaction:
		// Gross Card Sales 110,000 KRW (Supply 100,000 + VAT 10,000)
		// Card Receivable (Debit 1020): 110,000
		// Product Sales (Credit 4010): 100,000
		// Output VAT (Credit 2020): 10,000
		lines := []JournalLine{
			{AccountCode: "1020", Debit: decimal.RequireFromString("110000"), Credit: decimal.Zero, Description: "신한카드 승인 매출"},
			{AccountCode: "4010", Debit: decimal.Zero, Credit: decimal.RequireFromString("100000"), Description: "식사 상품 매출"},
			{AccountCode: "2020", Debit: decimal.Zero, Credit: decimal.RequireFromString("10000"), Description: "매출 부가가치세 10%"},
		}

		entry, err := svc.PostJournalEntry(ctx, storeID, now, "카드 매출 발생", lines)
		if err != nil {
			t.Fatalf("failed to post journal entry: %v", err)
		}

		if entry.EntryNumber == "" {
			t.Errorf("expected generated entry number")
		}

		// Also record Food Ingredients COGS (Debit 5010: 30,000, Credit 1010: 30,000)
		cogsLines := []JournalLine{
			{AccountCode: "5010", Debit: decimal.RequireFromString("30000"), Credit: decimal.Zero, Description: "식자재 매입"},
			{AccountCode: "1010", Debit: decimal.Zero, Credit: decimal.RequireFromString("30000"), Description: "보통예금 결제"},
		}
		_, err = svc.PostJournalEntry(ctx, storeID, now, "식자재 결제", cogsLines)
		if err != nil {
			t.Fatalf("failed to post cogs: %v", err)
		}

		// Also record Card Fee (Debit 5020: 1,650, Credit 1020: 1,650)
		feeLines := []JournalLine{
			{AccountCode: "5020", Debit: decimal.RequireFromString("1650"), Credit: decimal.Zero, Description: "카드사 수수료 1.5%"},
			{AccountCode: "1020", Debit: decimal.Zero, Credit: decimal.RequireFromString("1650"), Description: "카드미수금 차감"},
		}
		_, err = svc.PostJournalEntry(ctx, storeID, now, "카드 수수료 차감", feeLines)
		if err != nil {
			t.Fatalf("failed to post fee: %v", err)
		}

		// Calculate P&L
		pnl, err := svc.CalculatePnL(ctx, storeID, now.AddDate(0, 0, -1), now.AddDate(0, 0, 1))
		if err != nil {
			t.Fatalf("failed to calculate PnL: %v", err)
		}

		// Gross Sales: 100,000 + 10,000 = 110,000
		if !pnl.NetSales.Equal(decimal.RequireFromString("100000")) {
			t.Errorf("expected Net Sales 100000, got %s", pnl.NetSales)
		}
		// Gross Profit: Net Sales (100,000) - COGS (30,000) = 70,000
		if !pnl.GrossProfit.Equal(decimal.RequireFromString("70000")) {
			t.Errorf("expected Gross Profit 70000, got %s", pnl.GrossProfit)
		}
		// Contribution Margin: Gross Profit (70,000) - Fee (1,650) = 68,350
		if !pnl.ContributionMargin.Equal(decimal.RequireFromString("68350")) {
			t.Errorf("expected Contribution Margin 68350, got %s", pnl.ContributionMargin)
		}
	})

	t.Run("Unbalanced entry is strictly rejected (Debit != Credit invariant)", func(t *testing.T) {
		unbalancedLines := []JournalLine{
			{AccountCode: "1020", Debit: decimal.RequireFromString("100000"), Credit: decimal.Zero},
			{AccountCode: "4010", Debit: decimal.Zero, Credit: decimal.RequireFromString("90000")}, // 10,000 mismatch!
		}

		_, err := svc.PostJournalEntry(ctx, storeID, now, "불일치 분개", unbalancedLines)
		if err == nil {
			t.Fatalf("unbalanced journal entry must be rejected")
		}
	})

	t.Run("Immutable correction via reversal entry", func(t *testing.T) {
		lines := []JournalLine{
			{AccountCode: "1040", Debit: decimal.RequireFromString("50000"), Credit: decimal.Zero},
			{AccountCode: "4010", Debit: decimal.Zero, Credit: decimal.RequireFromString("50000")},
		}
		entry, err := svc.PostJournalEntry(ctx, storeID, now, "오기입된 현금매출", lines)
		if err != nil {
			t.Fatalf("failed to post: %v", err)
		}

		// Reverse it
		reversal, err := svc.ReverseJournalEntry(ctx, storeID, entry.ID, "금액 오기입 취소")
		if err != nil {
			t.Fatalf("failed to reverse: %v", err)
		}

		if len(reversal.Lines) != 2 {
			t.Errorf("expected 2 reversal lines")
		}
		// Check that Debit and Credit were flipped
		if !reversal.Lines[0].Credit.Equal(decimal.RequireFromString("50000")) {
			t.Errorf("expected first line credit to be 50000")
		}

		// Re-reversing must fail
		_, err = svc.ReverseJournalEntry(ctx, storeID, entry.ID, "중복 취소 시도")
		if err == nil {
			t.Fatalf("double reversal must be rejected")
		}
	})
}
