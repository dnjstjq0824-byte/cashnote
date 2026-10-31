package expense

import (
	"context"
	"regexp"
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
	ProofTaxInvoice  ProofType = "TAX_INVOICE"    // 전자세금계산서
	ProofCashReceipt ProofType = "CASH_RECEIPT"   // 지출증빙용 현금영수증
	ProofCreditCard  ProofType = "CREDIT_CARD"    // 사업용 신용카드
	ProofSimple      ProofType = "SIMPLE_RECEIPT" // 간이영수증 (3만원 한도)
)

// Vendor represents a business partner/supplier
type Vendor struct {
	ID             uuid.UUID `json:"id"`
	StoreID        uuid.UUID `json:"store_id"`
	BusinessNumber string    `json:"business_number"` // 10-digit BRN
	TradeName      string    `json:"trade_name"`      // 상호명
	Representative string    `json:"representative"`  // 대표자명
	Category       string    `json:"category"`        // 업태/종목 (식자재, 임대인, 세무사 등)
	Phone          string    `json:"phone"`
	Memo           string    `json:"memo"`
	CreatedAt      time.Time `json:"created_at"`
}

// Expense represents single business expense item
type Expense struct {
	ID            uuid.UUID       `json:"id"`
	StoreID       uuid.UUID       `json:"store_id"`
	VendorID      *uuid.UUID      `json:"vendor_id,omitempty"`
	ExpenseDate   time.Time       `json:"expense_date"`
	CategoryCode  CategoryCode    `json:"category_code"`
	VendorName    string          `json:"vendor_name"`
	SupplyAmount  decimal.Decimal `json:"supply_amount"`
	VatAmount     decimal.Decimal `json:"vat_amount"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
	PaymentMethod string          `json:"payment_method"` // BANK_TRANSFER, CARD, CASH
	ProofType     ProofType       `json:"proof_type"`
	Memo          string          `json:"memo"`
	SourceLevel   string          `json:"source_level"` // LEVEL_1_MANUAL, LEVEL_2_FILE, LEVEL_3_API
	CreatedAt     time.Time       `json:"created_at"`
}

// Service manages expenses and vendors
type Service interface {
	CreateExpense(ctx context.Context, exp *Expense) error
	ListExpenses(ctx context.Context, storeID uuid.UUID, fromDate, toDate time.Time, category *CategoryCode) ([]*Expense, error)
	GetExpenseSummary(ctx context.Context, storeID uuid.UUID, fromDate, toDate time.Time) (decimal.Decimal, decimal.Decimal, error)
	CreateVendor(ctx context.Context, v *Vendor) error
	ListVendors(ctx context.Context, storeID uuid.UUID) ([]*Vendor, error)
	GetVendor(ctx context.Context, storeID, vendorID uuid.UUID) (*Vendor, error)
}

type serviceImpl struct {
	mu       sync.RWMutex
	expenses map[uuid.UUID][]*Expense
	vendors  map[uuid.UUID][]*Vendor
}

func NewService() Service {
	return &serviceImpl{
		expenses: make(map[uuid.UUID][]*Expense),
		vendors:  make(map[uuid.UUID][]*Vendor),
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

	if exp.SourceLevel == "" {
		exp.SourceLevel = "LEVEL_1_MANUAL"
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

	items := s.expenses[storeID]
	fixed := decimal.Zero
	variable := decimal.Zero

	for _, exp := range items {
		if !fromDate.IsZero() && exp.ExpenseDate.Before(fromDate) {
			continue
		}
		if !toDate.IsZero() && exp.ExpenseDate.After(toDate) {
			continue
		}

		switch exp.CategoryCode {
		case CategoryFixedRent, CategoryFixedLabor, CategoryUtility:
			fixed = fixed.Add(exp.TotalAmount)
		case CategoryVariableIngredient, CategoryVariablePackaging, CategoryOther:
			variable = variable.Add(exp.TotalAmount)
		}
	}

	return fixed, variable, nil
}

var brnRegex = regexp.MustCompile(`^[0-9]{10}$`)

func (s *serviceImpl) CreateVendor(ctx context.Context, v *Vendor) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if v.TradeName == "" {
		return apperr.NewBadRequest("상호명은 필수 입력 항목입니다.")
	}

	cleanBRN := regexp.MustCompile(`[^0-9]`).ReplaceAllString(v.BusinessNumber, "")
	if cleanBRN != "" && !brnRegex.MatchString(cleanBRN) {
		return apperr.NewBadRequest("사업자등록번호는 10자리 숫자여야 합니다.")
	}
	v.BusinessNumber = cleanBRN

	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	v.CreatedAt = time.Now().UTC()

	s.vendors[v.StoreID] = append(s.vendors[v.StoreID], v)
	return nil
}

func (s *serviceImpl) ListVendors(ctx context.Context, storeID uuid.UUID) ([]*Vendor, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.vendors[storeID], nil
}

func (s *serviceImpl) GetVendor(ctx context.Context, storeID, vendorID uuid.UUID) (*Vendor, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, v := range s.vendors[storeID] {
		if v.ID == vendorID {
			return v, nil
		}
	}
	return nil, apperr.NewNotFound("거래처")
}
