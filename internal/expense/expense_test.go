package expense_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"merchant-os/internal/expense"
)

func TestVendorManagement(t *testing.T) {
	svc := expense.NewService()
	ctx := context.Background()
	storeID := uuid.New()

	// 1. 유효하지 않은 사업자번호 (9자리) -> 에러 기대
	invalidVendor := &expense.Vendor{
		StoreID:        storeID,
		BusinessNumber: "123456789",
		TradeName:      "잘못된식자재",
	}
	err := svc.CreateVendor(ctx, invalidVendor)
	if err == nil {
		t.Fatal("10자리가 아닌 사업자번호는 거부되어야 합니다.")
	}

	// 2. 정상 사업자번호 (하이픈 포함 10자리) -> 정규화되어 등록 성공
	validVendor := &expense.Vendor{
		StoreID:        storeID,
		BusinessNumber: "123-45-67890",
		TradeName:      "(주)농산물유통",
		Representative: "홍길동",
		Category:       "식자재도매",
		Phone:          "010-1234-5678",
	}
	err = svc.CreateVendor(ctx, validVendor)
	if err != nil {
		t.Fatalf("거래처 등록 실패: %v", err)
	}

	if validVendor.BusinessNumber != "1234567890" {
		t.Errorf("사업자등록번호 하이픈 정규화 실패: %s", validVendor.BusinessNumber)
	}

	// 3. 거래처 조회
	v, err := svc.GetVendor(ctx, storeID, validVendor.ID)
	if err != nil {
		t.Fatalf("거래처 단건 조회 실패: %v", err)
	}
	if v.TradeName != "(주)농산물유통" {
		t.Errorf("조회된 상호명 불일치: %s", v.TradeName)
	}
}

func TestExpenseSummaryFixedAndVariable(t *testing.T) {
	svc := expense.NewService()
	ctx := context.Background()
	storeID := uuid.New()
	now := time.Now()

	// 고정비 1: 임차료 1,500,000원 + 부가세 150,000원 = 1,650,000원
	_ = svc.CreateExpense(ctx, &expense.Expense{
		StoreID:       storeID,
		ExpenseDate:   now,
		CategoryCode:  expense.CategoryFixedRent,
		VendorName:    "건물주",
		SupplyAmount:  decimal.NewFromInt(1500000),
		VatAmount:     decimal.NewFromInt(150000),
		PaymentMethod: "BANK_TRANSFER",
		ProofType:     expense.ProofTaxInvoice,
	})

	// 고정비 2: 정규직 급여 2,500,000원
	_ = svc.CreateExpense(ctx, &expense.Expense{
		StoreID:       storeID,
		ExpenseDate:   now,
		CategoryCode:  expense.CategoryFixedLabor,
		VendorName:    "주방장",
		SupplyAmount:  decimal.NewFromInt(2500000),
		VatAmount:     decimal.Zero,
		PaymentMethod: "BANK_TRANSFER",
		ProofType:     expense.ProofSimple,
	})

	// 변동비 1: 식자재 800,000원 + 부가세 80,000원 = 880,000원
	_ = svc.CreateExpense(ctx, &expense.Expense{
		StoreID:       storeID,
		ExpenseDate:   now,
		CategoryCode:  expense.CategoryVariableIngredient,
		VendorName:    "가락시장",
		SupplyAmount:  decimal.NewFromInt(800000),
		VatAmount:     decimal.NewFromInt(80000),
		PaymentMethod: "CARD",
		ProofType:     expense.ProofCreditCard,
	})

	fixed, variable, err := svc.GetExpenseSummary(ctx, storeID, time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("비용 요약 조회 실패: %v", err)
	}

	expectedFixed := decimal.NewFromInt(4150000) // 165만 + 250만
	expectedVariable := decimal.NewFromInt(880000)

	if !fixed.Equal(expectedFixed) {
		t.Errorf("고정비 불일치: 기대값 %s, 실제값 %s", expectedFixed, fixed)
	}
	if !variable.Equal(expectedVariable) {
		t.Errorf("변동비 불일치: 기대값 %s, 실제값 %s", expectedVariable, variable)
	}
}
