package tax

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"merchant-os/internal/business"
)

func TestTaxService(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	storeID := uuid.New()

	t.Run("General taxable VAT estimate calculation", func(t *testing.T) {
		salesSupply := decimal.RequireFromString("100000000") // 1억 (매출세액 1천만원)
		purchaseVat := decimal.RequireFromString("4000000")   // 매입세액 400만원
		cardSales := decimal.RequireFromString("80000000")    // 카드매출 8천만원 -> 1.3% = 104만원
		agriPurchase := decimal.RequireFromString("10900000") // 면세 농산물 1090만원 -> 9/109 = 90만원

		report, err := svc.EstimateVat(ctx, storeID, business.TaxTypeGeneralTaxable, 2026, Term1Final, salesSupply, purchaseVat, cardSales, agriPurchase, 15)
		if err != nil {
			t.Fatalf("failed to calculate VAT: %v", err)
		}

		// Output VAT: 10,000,000
		if !report.OutputVat.Equal(decimal.RequireFromString("10000000")) {
			t.Errorf("expected output VAT 10,000,000, got %s", report.OutputVat)
		}

		// CC Issue Deduction: 80,000,000 * 0.013 = 1,040,000
		expectedCC := decimal.RequireFromString("1040000")
		if !report.CreditCardIssueDeduction.Equal(expectedCC) {
			t.Errorf("expected CC deduction %s, got %s", expectedCC, report.CreditCardIssueDeduction)
		}

		// Deemed input VAT: 10,900,000 * 9 / 109 = 900,000
		expectedDeemed := decimal.RequireFromString("900000")
		if !report.DeemedInputVatDeduction.Equal(expectedDeemed) {
			t.Errorf("expected deemed input VAT %s, got %s", expectedDeemed, report.DeemedInputVatDeduction)
		}

		// Estimated Payable: 10,000,000 - (4,000,000 + 1,040,000 + 900,000 = 5,940,000) = 4,060,000
		expectedPayable := decimal.RequireFromString("4060000")
		if !report.EstimatedPayableVat.Equal(expectedPayable) {
			t.Errorf("expected payable %s, got %s", expectedPayable, report.EstimatedPayableVat)
		}
	})

	t.Run("Simplified taxable exemption below threshold", func(t *testing.T) {
		salesSupply := decimal.RequireFromString("20000000") // 2천만원 (공급대가 2,200만원 < 4,800만원)
		report, err := svc.EstimateVat(ctx, storeID, business.TaxTypeSimplifiedTaxable, 2026, Term1Final, salesSupply, decimal.Zero, decimal.Zero, decimal.Zero, 2)
		if err != nil {
			t.Fatalf("failed to calculate simplified VAT: %v", err)
		}

		if !report.SimplifiedExemptionApplies {
			t.Errorf("expected simplified exemption to apply")
		}
		if !report.EstimatedPayableVat.IsZero() {
			t.Errorf("expected 0 payable for exempt simplified merchant")
		}
	})
}

