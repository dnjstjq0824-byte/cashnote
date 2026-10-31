package ledger_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"merchant-os/internal/ledger"
)

func TestStandardChartOfAccounts(t *testing.T) {
	svc := ledger.NewService()
	coa := svc.GetChartOfAccounts()

	if len(coa) < 40 {
		t.Fatalf("기대된 표준 계정과목 수(40개 이상)보다 적습니다: %d", len(coa))
	}

	foundCash := false
	foundSales := false
	foundFoodCost := false

	for _, acc := range coa {
		if acc.Code == "1010" && acc.Name == "보통예금(은행계좌)" {
			foundCash = true
		}
		if acc.Code == "4010" && acc.Name == "상품/음식매출" {
			foundSales = true
		}
		if acc.Code == "5010" && acc.Name == "식자재/원재료매입(매출원가)" {
			foundFoodCost = true
		}
	}

	if !foundCash || !foundSales || !foundFoodCost {
		t.Errorf("필수 핵심 계정과목이 누락되었습니다: cash=%v, sales=%v, food=%v", foundCash, foundSales, foundFoodCost)
	}
}

func TestPostJournalEntry_Unbalanced(t *testing.T) {
	svc := ledger.NewService()
	ctx := context.Background()
	storeID := uuid.New()

	// 차변 10,000 / 대변 9,000 -> 대차 불일치 에러 기대
	lines := []ledger.JournalLine{
		{AccountCode: "1010", Debit: decimal.NewFromInt(10000), Credit: decimal.Zero},
		{AccountCode: "4010", Debit: decimal.Zero, Credit: decimal.NewFromInt(9000)},
	}

	_, err := svc.PostJournalEntry(ctx, storeID, time.Now(), "대차불일치 테스트", "LEVEL_1_MANUAL", lines)
	if err == nil {
		t.Fatal("대차불일치 전표에 대해 에러가 발생해야 합니다.")
	}
}

func TestPostManualExpenseAndRevenue(t *testing.T) {
	svc := ledger.NewService()
	ctx := context.Background()
	storeID := uuid.New()
	now := time.Now()

	// 1. 수기 매출 등록: 음식매출 100,000원 + 부가세 10,000원 -> 보통예금(1010) 입금
	revInput := ledger.ManualRevenueInput{
		RevenueDate:   now,
		Channel:       "STORE",
		PaymentMethod: "BANK",
		SupplyAmount:  decimal.NewFromInt(100000),
		VatAmount:     decimal.NewFromInt(10000),
		Description:   "테이블 홀 매출",
	}
	revEntry, err := svc.PostManualRevenue(ctx, storeID, revInput)
	if err != nil {
		t.Fatalf("수기 매출 전표 등록 실패: %v", err)
	}
	if revEntry.IsReversed {
		t.Error("초기 생성 전표는 IsReversed가 false여야 합니다.")
	}

	// 2. 수기 비용 등록: 식자재 30,000원 + 부가세 3,000원 -> 보통예금(1010) 출금
	expInput := ledger.ManualExpenseInput{
		ExpenseDate:   now,
		AccountCode:   "5010",
		PaymentMethod: "BANK",
		SupplyAmount:  decimal.NewFromInt(30000),
		VatAmount:     decimal.NewFromInt(3000),
		VendorName:    "신선야채도매",
		Description:   "양파 및 대파 구매",
	}
	_, err = svc.PostManualExpense(ctx, storeID, expInput)
	if err != nil {
		t.Fatalf("수기 비용 전표 등록 실패: %v", err)
	}

	// 3. P&L 산출 검증
	pnl, err := svc.CalculatePnL(ctx, storeID, time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("손익계산서 산출 실패: %v", err)
	}

	expectedNetSales := decimal.NewFromInt(100000)
	expectedCOGS := decimal.NewFromInt(30000)
	expectedGrossProfit := decimal.NewFromInt(70000)

	if !pnl.NetSales.Equal(expectedNetSales) {
		t.Errorf("순매출액 불일치: 기대값 %s, 실제값 %s", expectedNetSales, pnl.NetSales)
	}
	if !pnl.COGS.Equal(expectedCOGS) {
		t.Errorf("매출원가 불일치: 기대값 %s, 실제값 %s", expectedCOGS, pnl.COGS)
	}
	if !pnl.GrossProfit.Equal(expectedGrossProfit) {
		t.Errorf("매출총이익 불일치: 기대값 %s, 실제값 %s", expectedGrossProfit, pnl.GrossProfit)
	}

	// 4. 합계잔액시산표(Trial Balance) 대차 일치 검증
	tb, err := svc.GetTrialBalance(ctx, storeID, time.Time{})
	if err != nil {
		t.Fatalf("시산표 산출 실패: %v", err)
	}
	if !tb.IsBalanced {
		t.Errorf("시산표 대차 불일치: 차변합계 %s != 대변합계 %s", tb.TotalDebitBal, tb.TotalCreditBal)
	}
}

func TestAccountingPeriodLockAndReversal(t *testing.T) {
	svc := ledger.NewService()
	ctx := context.Background()
	storeID := uuid.New()
	userID := uuid.New()

	dateInJan := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	// 1월 전표 등록
	lines := []ledger.JournalLine{
		{AccountCode: "1010", Debit: decimal.NewFromInt(50000), Credit: decimal.Zero, Description: "입금"},
		{AccountCode: "4010", Debit: decimal.Zero, Credit: decimal.NewFromInt(50000), Description: "매출"},
	}
	entry, err := svc.PostJournalEntry(ctx, storeID, dateInJan, "1월 정상 매출", "LEVEL_1_MANUAL", lines)
	if err != nil {
		t.Fatalf("전표 등록 실패: %v", err)
	}

	// 2026년 1월 회계기간 마감(LOCK)
	period, err := svc.ClosePeriod(ctx, storeID, 2026, 1, userID)
	if err != nil {
		t.Fatalf("회계기간 마감 실패: %v", err)
	}
	if period.Status != ledger.PeriodLocked {
		t.Errorf("마감 후 상태는 LOCKED여야 합니다: %s", period.Status)
	}

	// 마감된 1월에 추가 전표 등록 시도 -> 실패해야 함
	_, err = svc.PostJournalEntry(ctx, storeID, dateInJan, "마감 후 불법 전표", "LEVEL_1_MANUAL", lines)
	if err == nil {
		t.Fatal("마감된 기간(LOCKED)에 전표 작성이 허용되어서는 안 됩니다.")
	}

	// 기존 1월 전표 역분개(Reversal) -> 마감되었으므로 역분개 전표는 현재 시점에 반영되어 성공해야 함
	revEntry, err := svc.ReverseJournalEntry(ctx, storeID, entry.ID, "고객 결제 취소 요청")
	if err != nil {
		t.Fatalf("역분개 실패: %v", err)
	}
	if !entry.IsReversed {
		t.Error("원본 전표의 IsReversed가 true로 갱신되어야 합니다.")
	}
	if revEntry.EntryDate.Equal(dateInJan) {
		t.Error("마감된 기간의 전표 역분개는 현재 오픈 기간 날짜로 전기되어야 합니다.")
	}
}
