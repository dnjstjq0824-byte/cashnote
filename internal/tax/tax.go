package tax

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"merchant-os/internal/business"
)

// VatTerm represents Korean VAT filing term
type VatTerm string

const (
	Term1Preliminary VatTerm = "1_PRELIMINARY" // 1기 예정 (1월~3월, 신고: 4/25)
	Term1Final       VatTerm = "1_FINAL"       // 1기 확정 (1월~6월 또는 4월~6월, 신고: 7/25)
	Term2Preliminary VatTerm = "2_PRELIMINARY" // 2기 예정 (7월~9월, 신고: 10/25)
	Term2Final       VatTerm = "2_FINAL"       // 2기 확정 (7월~12월 또는 10월~12월, 신고: 1/25)
)

// VatEstimateReport represents calculated VAT estimation breakdown
type VatEstimateReport struct {
	StoreID                     uuid.UUID                `json:"store_id"`
	Year                        int                      `json:"year"`
	Term                        VatTerm                  `json:"term"`
	TaxType                     business.BusinessTaxType `json:"tax_type"`
	TaxableSalesSupply          decimal.Decimal          `json:"taxable_sales_supply"`
	OutputVat                   decimal.Decimal          `json:"output_vat"`
	DeductiblePurchaseVat       decimal.Decimal          `json:"deductible_purchase_vat"`
	CreditCardIssueDeduction    decimal.Decimal          `json:"credit_card_issue_deduction"`
	DeemedInputVatDeduction     decimal.Decimal          `json:"deemed_input_vat_deduction"`
	EstimatedPayableVat         decimal.Decimal          `json:"estimated_payable_vat"`
	SimplifiedExemptionApplies  bool                     `json:"simplified_exemption_applies"`
	FilingDueDate               string                   `json:"filing_due_date"`
	DaysRemaining               int                      `json:"days_remaining"`
	EligibleInvoiceCount        int                      `json:"eligible_invoice_count"`
	MissingReceiptCount         int                      `json:"missing_receipt_count"`
	RecommendationMemo          string                   `json:"recommendation_memo"`
}

// Service defines tax and VAT estimation operations
type Service interface {
	EstimateVat(ctx context.Context, storeID uuid.UUID, taxType business.BusinessTaxType, year int, term VatTerm, salesSupply, purchaseVat, cardSales, agriPurchase decimal.Decimal, invoiceCount int) (*VatEstimateReport, error)
}

type serviceImpl struct{}

func NewService() Service {
	return &serviceImpl{}
}

func (s *serviceImpl) EstimateVat(ctx context.Context, storeID uuid.UUID, taxType business.BusinessTaxType, year int, term VatTerm, salesSupply, purchaseVat, cardSales, agriPurchase decimal.Decimal, invoiceCount int) (*VatEstimateReport, error) {
	outputVat := decimal.Zero
	ccDeduction := decimal.Zero
	deemedVat := decimal.Zero
	estimatedPayable := decimal.Zero
	simplifiedExemption := false

	now := time.Now().UTC()
	var dueDate time.Time

	switch term {
	case Term1Preliminary:
		dueDate = time.Date(year, 4, 25, 23, 59, 59, 0, time.UTC)
	case Term1Final:
		dueDate = time.Date(year, 7, 25, 23, 59, 59, 0, time.UTC)
	case Term2Preliminary:
		dueDate = time.Date(year, 10, 25, 23, 59, 59, 0, time.UTC)
	case Term2Final:
		dueDate = time.Date(year+1, 1, 25, 23, 59, 59, 0, time.UTC)
	default:
		dueDate = time.Date(year, 7, 25, 23, 59, 59, 0, time.UTC)
	}

	daysRemaining := int(dueDate.Sub(now).Hours() / 24)
	if daysRemaining < 0 {
		daysRemaining = 0
	}

	if taxType == business.TaxTypeSimplifiedTaxable {
		// 간이과세자: (공급대가 * 부가가치율 15~40% * 10%) - 공제세액(0.5%)
		// 업종별 부가가치율: 음식점업/소매업 15% 기준
		vatRate := decimal.NewFromFloat(0.15)
		supplyPlusVat := salesSupply.Mul(decimal.NewFromFloat(1.1)) // 공급대가

		// 납부의무 면제 기준: 연간 공급대가 4,800만원 미만 시
		if supplyPlusVat.LessThan(decimal.RequireFromString("48000000")) {
			simplifiedExemption = true
			estimatedPayable = decimal.Zero
		} else {
			rawVat := supplyPlusVat.Mul(vatRate).Mul(decimal.NewFromFloat(0.1)).Floor()
			deduction := purchaseVat.Mul(decimal.NewFromFloat(0.005)).Floor()
			estimatedPayable = rawVat.Sub(deduction)
			if estimatedPayable.IsNegative() {
				estimatedPayable = decimal.Zero
			}
		}
	} else {
		// 일반과세자: 매출세액(10%) - 매입세액 - 신용카드발행공제(1.3%, 한도 1000만원) - 의제매입세액공제(9/109)
		outputVat = salesSupply.Mul(decimal.NewFromFloat(0.1)).Floor()

		// 1. 신용카드매출전표 발행세액공제 1.3% (연간 1천만원 한도)
		if !cardSales.IsZero() {
			rawCC := cardSales.Mul(decimal.NewFromFloat(0.013)).Floor()
			maxLimit := decimal.RequireFromString("10000000")
			if rawCC.GreaterThan(maxLimit) {
				ccDeduction = maxLimit
			} else {
				ccDeduction = rawCC
			}
		}

		// 2. 의제매입세액공제 (음식점업 면세 농산물 9/109)
		if !agriPurchase.IsZero() {
			deemedVat = agriPurchase.Mul(decimal.NewFromInt(9)).Div(decimal.NewFromInt(109)).Floor()
		}

		totalDeductions := purchaseVat.Add(ccDeduction).Add(deemedVat)
		estimatedPayable = outputVat.Sub(totalDeductions)
		if estimatedPayable.IsNegative() {
			estimatedPayable = decimal.Zero // 환급 또는 0원
		}
	}

	memo := fmt.Sprintf("신고기한까지 약 %d일 남았습니다. 전자세금계산서 %d건 정상 집계 완료.", daysRemaining, invoiceCount)
	if simplifiedExemption {
		memo = "연간 공급대가 4,800만원 미만으로 부가가치세 납부의무 면제 대상 후보입니다."
	}

	return &VatEstimateReport{
		StoreID:                    storeID,
		Year:                       year,
		Term:                       term,
		TaxType:                    taxType,
		TaxableSalesSupply:         salesSupply,
		OutputVat:                  outputVat,
		DeductiblePurchaseVat:      purchaseVat,
		CreditCardIssueDeduction:   ccDeduction,
		DeemedInputVatDeduction:    deemedVat,
		EstimatedPayableVat:        estimatedPayable,
		SimplifiedExemptionApplies: simplifiedExemption,
		FilingDueDate:              dueDate.Format("2006-01-25"),
		DaysRemaining:              daysRemaining,
		EligibleInvoiceCount:       invoiceCount,
		MissingReceiptCount:        0,
		RecommendationMemo:         memo,
	}, nil
}

