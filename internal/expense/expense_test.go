package expense

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestExpenseService(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	storeID := uuid.New()
	now := time.Now()

	t.Run("Create expense and check fixed vs variable summary", func(t *testing.T) {
		// Fixed Rent Expense
		exp1 := &Expense{
			StoreID:      storeID,
			ExpenseDate:  now,
			CategoryCode: CategoryFixedRent,
			VendorName:   "건물관리단",
			SupplyAmount: decimal.RequireFromString("2000000"),
			VatAmount:    decimal.RequireFromString("200000"),
			ProofType:    ProofTaxInvoice,
		}
		err := svc.CreateExpense(ctx, exp1)
		if err != nil {
			t.Fatalf("failed to create rent expense: %v", err)
		}

		if !exp1.TotalAmount.Equal(decimal.RequireFromString("2200000")) {
			t.Errorf("expected total amount 2,200,000, got %s", exp1.TotalAmount)
		}

		// Variable Ingredient Expense
		exp2 := &Expense{
			StoreID:      storeID,
			ExpenseDate:  now,
			CategoryCode: CategoryVariableIngredient,
			VendorName:   "대박유통",
			SupplyAmount: decimal.RequireFromString("1500000"),
			VatAmount:    decimal.RequireFromString("150000"),
			ProofType:    ProofTaxInvoice,
		}
		err = svc.CreateExpense(ctx, exp2)
		if err != nil {
			t.Fatalf("failed to create ingredient expense: %v", err)
		}

		fixed, variable, err := svc.GetExpenseSummary(ctx, storeID, time.Time{}, time.Time{})
		if err != nil {
			t.Fatalf("failed to get expense summary: %v", err)
		}

		if !fixed.Equal(decimal.RequireFromString("2200000")) {
			t.Errorf("expected fixed expense 2,200,000, got %s", fixed)
		}
		if !variable.Equal(decimal.RequireFromString("1650000")) {
			t.Errorf("expected variable expense 1,650,000, got %s", variable)
		}
	})
}
