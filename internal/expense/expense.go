package expense

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	apperr "merchant-os/internal/platform/errors"
)

// CategoryCode defines expense category codes
type CategoryCode string

const (
	CategoryFixedRent          CategoryCode = "FIXED_RENT"
	CategoryFixedLabor         CategoryCode = "FIXED_LABOR"
	CategoryVariableIngredient CategoryCode = "VARIABLE_INGREDIENTS"
	CategoryVariablePackaging  CategoryCode = "VARIABLE_PACKAGING"
	CategoryUtility            CategoryCode = "UTILITY"
	CategoryOther              CategoryCode = "OTHER"
)

// ProofType defines tax-deductible proof documentation
type ProofType string

const (
	ProofTaxInvoice  ProofType = "TAX_INVOICE"   // 전자세금계산서
	ProofCashReceipt ProofType = "CASH_RECEIPT"  // 지출증빙용 현금영수증
	ProofCreditCard  ProofType = "CREDIT_CARD"   // 사업용 신용카드
	ProofSimple      ProofType = "SIMPLE_RECEIPT"// 간이영수증 (3만원 한도)
)

// Expense represents single business expense item
type Expense struct {
	ID            uuid.UUID       `json:"id"`
	StoreID       uuid.UUID       `json:"store_id"`
	ExpenseDate   time.Time       `json:"expense_date"`
	CategoryCode  CategoryCode    `json:"category_code"`
	VendorName    string          `json:"vendor_name"`
	SupplyAmount  decimal.Decimal `json:"supply_amount"`
	VatAmount     decimal.Decimal `json:"vat_amount"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
	PaymentMethod string          `json:"payment_method"` // BANK_TRANSFER, CARD, CASH
	ProofType     ProofType       `json:"proof_type"`
	Memo          string          `json:"memo"`
	CreatedAt     time.Time       `json:"created_at"`
}

// Service manages expenses
type Service interface {
	CreateExpense(ctx context.Context, exp *Expense) error
	ListExpenses(ctx context.Context, storeID uuid.UUID, fromDate, toDate time.Time, category *CategoryCode) ([]*Expense, error)
	GetExpenseSummary(ctx context.Context, storeID uuid.UUID, fromDate, toDate time.Time) (decimal.Decimal, decimal.Decimal, error)
}

type serviceImpl struct {
	mu       sync.RWMutex
	expenses map[uuid.UUID][]*Expense
}

func NewService() Service {
	return &serviceImpl{
		expenses: make(map[uuid.UUID][]*Expense),
	}
}

func (s *serviceImpl) CreateExpense(ctx context.Context, exp *Expense) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if exp.ID == uuid.Nil {
		exp.ID = uuid.New()
	}

	if exp.SupplyAmount.IsNegative() || exp.VatAmount.IsNegative() {
		return apperr.NewBadRequest("비용 금액은 0 이상이어야 합니다.")
	}

	exp.TotalAmount = exp.SupplyAmount.Add(exp.VatAmount)
	exp.CreatedAt = time.Now().UTC()

	s.expenses[exp.StoreID] = append(s.expenses[exp.StoreID], exp)
	return nil
}

func (s *serviceImpl) ListExpenses(ctx context.Context, storeID uuid.UUID, fromDate, toDate time.Time, category *CategoryCode) ([]*Expense, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := s.expenses[storeID]
	var result []*Expense

	for _, exp := range items {
		if !fromDate.IsZero() && exp.ExpenseDate.Before(fromDate) {
			continue
		}
		if !toDate.IsZero() && exp.ExpenseDate.After(toDate) {
			continue
		}
		if category != nil && exp.CategoryCode != *category {
			continue
		}
		result = append(result, exp)
	}

	return result, nil
}

func (s *serviceImpl) GetExpenseSummary(ctx context.Context, storeID uuid.UUID, fromDate, toDate time.Time) (decimal.Decimal, decimal.Decimal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fixedTotal := decimal.Zero
	variableTotal := decimal.Zero

	for _, exp := range s.expenses[storeID] {
		if !fromDate.IsZero() && exp.ExpenseDate.Before(fromDate) {
			continue
		}
		if !toDate.IsZero() && exp.ExpenseDate.After(toDate) {
			continue
		}

		switch exp.CategoryCode {
		case CategoryFixedRent, CategoryFixedLabor, CategoryUtility:
			fixedTotal = fixedTotal.Add(exp.TotalAmount)
		default:
			variableTotal = variableTotal.Add(exp.TotalAmount)
		}
	}

	return fixedTotal, variableTotal, nil
}
