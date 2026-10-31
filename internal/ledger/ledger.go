package ledger

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	apperr "merchant-os/internal/platform/errors"
)

// AccountType represents standard accounting classification
type AccountType string

const (
	AccountAsset     AccountType = "ASSET"     // 자산 (차변 증가)
	AccountLiability AccountType = "LIABILITY" // 부채 (대변 증가)
	AccountEquity    AccountType = "EQUITY"    // 자본 (대변 증가)
	AccountRevenue   AccountType = "REVENUE"   // 수익 (대변 증가)
	AccountExpense   AccountType = "EXPENSE"   // 비용 (차변 증가)
)

// PeriodStatus represents accounting period status
type PeriodStatus string

const (
	PeriodOpen          PeriodStatus = "OPEN"
	PeriodClosingReview PeriodStatus = "CLOSING_REVIEW"
	PeriodLocked        PeriodStatus = "LOCKED"
)

// AccountingPeriod represents monthly/annual accounting closing period
type AccountingPeriod struct {
	ID        uuid.UUID    `json:"id"`
	StoreID   uuid.UUID    `json:"store_id"`
	Year      int          `json:"year"`
	Month     int          `json:"month"` // 1..12
	Status    PeriodStatus `json:"status"`
	ClosedAt  *time.Time   `json:"closed_at,omitempty"`
	ClosedBy  *uuid.UUID   `json:"closed_by,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}

// Account represents a Chart of Accounts entry
type Account struct {
	Code      string      `json:"code"`
	Name      string      `json:"name"`
	Type      AccountType `json:"type"`
	NormalBal string      `json:"normal_balance"` // DEBIT or CREDIT
	Category  string      `json:"category"`       // 유동자산, 비유동자산, 매출원가 등
}

// Standard Chart of Accounts (45 standard accounts for Korean SMBs)
var StandardCOA = []Account{
	// 1000s: Assets (자산)
	{Code: "1010", Name: "보통예금(은행계좌)", Type: AccountAsset, NormalBal: "DEBIT", Category: "당좌자산"},
	{Code: "1020", Name: "카드미수금(매입완료/입금예정)", Type: AccountAsset, NormalBal: "DEBIT", Category: "당좌자산"},
	{Code: "1030", Name: "배달앱미수금(정산예정)", Type: AccountAsset, NormalBal: "DEBIT", Category: "당좌자산"},
	{Code: "1040", Name: "현금시재", Type: AccountAsset, NormalBal: "DEBIT", Category: "당좌자산"},
	{Code: "1050", Name: "외상매출금(거래처외상)", Type: AccountAsset, NormalBal: "DEBIT", Category: "매출채권"},
	{Code: "1060", Name: "선급금(계약금선지급)", Type: AccountAsset, NormalBal: "DEBIT", Category: "당좌자산"},
	{Code: "1110", Name: "상품/식자재재고", Type: AccountAsset, NormalBal: "DEBIT", Category: "재고자산"},
	{Code: "1120", Name: "포장소모품재고", Type: AccountAsset, NormalBal: "DEBIT", Category: "재고자산"},
	{Code: "1210", Name: "임차보증금", Type: AccountAsset, NormalBal: "DEBIT", Category: "기타비유동자산"},
	{Code: "1220", Name: "인테리어/시설장치", Type: AccountAsset, NormalBal: "DEBIT", Category: "유형자산"},
	{Code: "1230", Name: "주방기구/비품", Type: AccountAsset, NormalBal: "DEBIT", Category: "유형자산"},
	{Code: "1240", Name: "POS/키오스크단말기", Type: AccountAsset, NormalBal: "DEBIT", Category: "유형자산"},
	{Code: "1290", Name: "감가상각누계액", Type: AccountAsset, NormalBal: "CREDIT", Category: "유형자산차감"},

	// 2000s: Liabilities (부채)
	{Code: "2010", Name: "외상매입금(식자재/비품미지급)", Type: AccountLiability, NormalBal: "CREDIT", Category: "매입채무"},
	{Code: "2020", Name: "부가세예수금(매출부가세)", Type: AccountLiability, NormalBal: "CREDIT", Category: "유동부채"},
	{Code: "2030", Name: "미지급금(일반비용/카드대금)", Type: AccountLiability, NormalBal: "CREDIT", Category: "유동부채"},
	{Code: "2040", Name: "예수금(4대보험/원천세)", Type: AccountLiability, NormalBal: "CREDIT", Category: "유동부채"},
	{Code: "2050", Name: "선수금(선불권/예치금)", Type: AccountLiability, NormalBal: "CREDIT", Category: "유동부채"},
	{Code: "2060", Name: "미지급급여(직원/알바)", Type: AccountLiability, NormalBal: "CREDIT", Category: "유동부채"},
	{Code: "2110", Name: "단기차입금(은행/소상공인대출)", Type: AccountLiability, NormalBal: "CREDIT", Category: "유동부채"},
	{Code: "2210", Name: "장기차입금(시설자금대출)", Type: AccountLiability, NormalBal: "CREDIT", Category: "비유동부채"},

	// 3000s: Equity (자본)
	{Code: "3010", Name: "자본금(초기창업투자)", Type: AccountEquity, NormalBal: "CREDIT", Category: "자본금"},
	{Code: "3020", Name: "인출금(대표자개인가져감)", Type: AccountEquity, NormalBal: "DEBIT", Category: "자본조정"},
	{Code: "3030", Name: "대표자입금(개인자금투입)", Type: AccountEquity, NormalBal: "CREDIT", Category: "자본잉여금"},
	{Code: "3040", Name: "이월이익잉여금", Type: AccountEquity, NormalBal: "CREDIT", Category: "이익잉여금"},

	// 4000s: Revenue (수익)
	{Code: "4010", Name: "상품/음식매출", Type: AccountRevenue, NormalBal: "CREDIT", Category: "영업수익"},
	{Code: "4020", Name: "배달매출", Type: AccountRevenue, NormalBal: "CREDIT", Category: "영업수익"},
	{Code: "4030", Name: "포장/테이크아웃매출", Type: AccountRevenue, NormalBal: "CREDIT", Category: "영업수익"},
	{Code: "4040", Name: "기타매출(음료/주류)", Type: AccountRevenue, NormalBal: "CREDIT", Category: "영업수익"},
	{Code: "4110", Name: "영업외수익(이자/지원금)", Type: AccountRevenue, NormalBal: "CREDIT", Category: "영업외수익"},

	// 5000s: Cost of Goods Sold & Expenses (비용)
	{Code: "5010", Name: "식자재/원재료매입(매출원가)", Type: AccountExpense, NormalBal: "DEBIT", Category: "매출원가"},
	{Code: "5020", Name: "카드결제/PG수수료", Type: AccountExpense, NormalBal: "DEBIT", Category: "판매수수료"},
	{Code: "5030", Name: "배달중개수수료및배달비", Type: AccountExpense, NormalBal: "DEBIT", Category: "판매수수료"},
	{Code: "5040", Name: "임차료및관리비", Type: AccountExpense, NormalBal: "DEBIT", Category: "고정비"},
	{Code: "5050", Name: "인건비및급여(정규직)", Type: AccountExpense, NormalBal: "DEBIT", Category: "고정비"},
	{Code: "5055", Name: "일용근로자/알바인건비", Type: AccountExpense, NormalBal: "DEBIT", Category: "변동인건비"},
	{Code: "5060", Name: "포장용기및소모품비", Type: AccountExpense, NormalBal: "DEBIT", Category: "변동비"},
	{Code: "5070", Name: "수도광열비(전기/가스/수도)", Type: AccountExpense, NormalBal: "DEBIT", Category: "고정관리비"},
	{Code: "5080", Name: "광고선전비(배달깃발/키워드)", Type: AccountExpense, NormalBal: "DEBIT", Category: "판매촉진비"},
	{Code: "5090", Name: "통신비및소프트웨어구독료", Type: AccountExpense, NormalBal: "DEBIT", Category: "일반관리비"},
	{Code: "5100", Name: "지급수수료(세무기장/방역/보안)", Type: AccountExpense, NormalBal: "DEBIT", Category: "일반관리비"},
	{Code: "5110", Name: "차량유지비(배달오토바이)", Type: AccountExpense, NormalBal: "DEBIT", Category: "차량유지비"},
	{Code: "5120", Name: "보험료(화재/배상책임)", Type: AccountExpense, NormalBal: "DEBIT", Category: "보험료"},
	{Code: "5130", Name: "감가상각비", Type: AccountExpense, NormalBal: "DEBIT", Category: "비현금비용"},
	{Code: "5210", Name: "영업외비용(대출이자/기타)", Type: AccountExpense, NormalBal: "DEBIT", Category: "영업외비용"},
}

// JournalLine represents a single debit or credit line
type JournalLine struct {
	AccountCode string          `json:"account_code"`
	Debit       decimal.Decimal `json:"debit"`
	Credit      decimal.Decimal `json:"credit"`
	Description string          `json:"description"`
}

// JournalEntry represents an immutable double-entry record
type JournalEntry struct {
	ID          uuid.UUID     `json:"id"`
	StoreID     uuid.UUID     `json:"store_id"`
	EntryNumber string        `json:"entry_number"`
	EntryDate   time.Time     `json:"entry_date"`
	Description string        `json:"description"`
	Lines       []JournalLine `json:"lines"`
	SourceLevel string        `json:"source_level"` // LEVEL_1_MANUAL, LEVEL_2_FILE, LEVEL_3_API, LEVEL_4_RECONCILED
	IsReversed  bool          `json:"is_reversed"`
	CreatedAt   time.Time     `json:"created_at"`
}

// TrialBalanceLine represents an aggregated account balance line in Trial Balance
type TrialBalanceLine struct {
	AccountCode string          `json:"account_code"`
	AccountName string          `json:"account_name"`
	AccountType AccountType     `json:"account_type"`
	DebitTotal  decimal.Decimal `json:"debit_total"`
	CreditTotal decimal.Decimal `json:"credit_total"`
	DebitBal    decimal.Decimal `json:"debit_balance"`
	CreditBal   decimal.Decimal `json:"credit_balance"`
}

// TrialBalance represents 합계잔액시산표
type TrialBalance struct {
	StoreID         uuid.UUID          `json:"store_id"`
	AsOfDate        time.Time          `json:"as_of_date"`
	Lines           []TrialBalanceLine `json:"lines"`
	TotalDebitBal   decimal.Decimal    `json:"total_debit_balance"`
	TotalCreditBal  decimal.Decimal    `json:"total_credit_balance"`
	IsBalanced      bool               `json:"is_balanced"`
}

// GeneralLedgerAccount represents specific account ledger with chronological transactions
type GeneralLedgerAccount struct {
	AccountCode  string          `json:"account_code"`
	AccountName  string          `json:"account_name"`
	AccountType  AccountType     `json:"account_type"`
	Entries      []GLEntryItem   `json:"entries"`
	TotalDebit   decimal.Decimal `json:"total_debit"`
	TotalCredit  decimal.Decimal `json:"total_credit"`
	FinalBalance decimal.Decimal `json:"final_balance"`
}

// GLEntryItem represents transaction detail in General Ledger
type GLEntryItem struct {
	EntryID     uuid.UUID       `json:"entry_id"`
	EntryNumber string          `json:"entry_number"`
	Date        time.Time       `json:"date"`
	Description string          `json:"description"`
	Debit       decimal.Decimal `json:"debit"`
	Credit      decimal.Decimal `json:"credit"`
	RunningBal  decimal.Decimal `json:"running_balance"`
}

// ProfitAndLossReport represents Korean Small Business 9-Layer P&L statement
type ProfitAndLossReport struct {
	StoreID             uuid.UUID       `json:"store_id"`
	GrossSales          decimal.Decimal `json:"gross_sales"`           // 1. 총매출액 (공급가+부가세)
	OutputVAT           decimal.Decimal `json:"output_vat"`            // 2. 부가세예수금
	NetSales            decimal.Decimal `json:"net_sales"`             // 3. 순매출액 (실질공급가액)
	COGS                decimal.Decimal `json:"cogs"`                  // 4. 매출원가 (식자재)
	GrossProfit         decimal.Decimal `json:"gross_profit"`          // 5. 매출총이익 (순매출 - 원가)
	PlatformFees        decimal.Decimal `json:"platform_fees"`         // 6. 플랫폼/카드수수료
	ContributionMargin  decimal.Decimal `json:"contribution_margin"`   // 7. 공헌이익 (매출총이익 - 수수료)
	FixedExpenses       decimal.Decimal `json:"fixed_expenses"`        // 8. 고정비 (임차료, 인건비, 공과금)
	OperatingProfit     decimal.Decimal `json:"operating_profit"`      // 9. 실질영업이익 (공헌이익 - 고정비)
	OperatingProfitRate decimal.Decimal `json:"operating_profit_rate"` // 영업이익률 (%)
}

// ManualExpenseInput represents Level 1 manual expense recording request
type ManualExpenseInput struct {
	ExpenseDate   time.Time       `json:"expense_date"`
	AccountCode   string          `json:"account_code"` // e.g. 5010 (식자재), 5040 (임차료)
	PaymentMethod string          `json:"payment_method"` // BANK (1010), CARD (2030), CASH (1040), PAYABLE (2010)
	SupplyAmount  decimal.Decimal `json:"supply_amount"`
	VatAmount     decimal.Decimal `json:"vat_amount"`
	VendorName    string          `json:"vendor_name"`
	Description   string          `json:"description"`
}

// ManualRevenueInput represents Level 1 manual revenue recording request
type ManualRevenueInput struct {
	RevenueDate   time.Time       `json:"revenue_date"`
	Channel       string          `json:"channel"` // STORE (4010), DELIVERY (4020), TAKEOUT (4030)
	PaymentMethod string          `json:"payment_method"` // CASH (1040), CARD_RECEIVABLE (1020), BANK (1010)
	SupplyAmount  decimal.Decimal `json:"supply_amount"`
	VatAmount     decimal.Decimal `json:"vat_amount"`
	Description   string          `json:"description"`
}

// Service defines double-entry ledger operations
type Service interface {
	GetChartOfAccounts() []Account
	PostJournalEntry(ctx context.Context, storeID uuid.UUID, entryDate time.Time, description, sourceLevel string, lines []JournalLine) (*JournalEntry, error)
	PostManualExpense(ctx context.Context, storeID uuid.UUID, input ManualExpenseInput) (*JournalEntry, error)
	PostManualRevenue(ctx context.Context, storeID uuid.UUID, input ManualRevenueInput) (*JournalEntry, error)
	ReverseJournalEntry(ctx context.Context, storeID, entryID uuid.UUID, reason string) (*JournalEntry, error)
	PostPriorPeriodAdjustment(ctx context.Context, storeID uuid.UUID, originalPeriodYear, originalPeriodMonth int, reason string, lines []JournalLine) (*JournalEntry, error)
	ClosePeriod(ctx context.Context, storeID uuid.UUID, year, month int, closedBy uuid.UUID) (*AccountingPeriod, error)
	GetPeriod(ctx context.Context, storeID uuid.UUID, year, month int) (*AccountingPeriod, error)
	CalculatePnL(ctx context.Context, storeID uuid.UUID, fromDate, toDate time.Time) (*ProfitAndLossReport, error)
	GetTrialBalance(ctx context.Context, storeID uuid.UUID, asOfDate time.Time) (*TrialBalance, error)
	GetGeneralLedger(ctx context.Context, storeID uuid.UUID, accountCode string, fromDate, toDate time.Time) (*GeneralLedgerAccount, error)
	ListEntries(ctx context.Context, storeID uuid.UUID) ([]*JournalEntry, error)
}

type serviceImpl struct {
	mu       sync.RWMutex
	accounts map[string]Account
	periods  map[string]*AccountingPeriod // key: storeID-YYYY-MM
	entries  map[uuid.UUID]*JournalEntry
	byStore  map[uuid.UUID][]*JournalEntry
}

func NewService() Service {
	accMap := make(map[string]Account)
	for _, acc := range StandardCOA {
		accMap[acc.Code] = acc
	}
	return &serviceImpl{
		accounts: accMap,
		periods:  make(map[string]*AccountingPeriod),
		entries:  make(map[uuid.UUID]*JournalEntry),
		byStore:  make(map[uuid.UUID][]*JournalEntry),
	}
}

func (s *serviceImpl) GetChartOfAccounts() []Account {
	return StandardCOA
}

func (s *serviceImpl) isPeriodLocked(storeID uuid.UUID, t time.Time) bool {
	key := fmt.Sprintf("%s-%04d-%02d", storeID.String(), t.Year(), t.Month())
	p, exists := s.periods[key]
	return exists && p.Status == PeriodLocked
}

func (s *serviceImpl) PostJournalEntry(ctx context.Context, storeID uuid.UUID, entryDate time.Time, description, sourceLevel string, lines []JournalLine) (*JournalEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Check accounting period lock
	if s.isPeriodLocked(storeID, entryDate) {
		return nil, apperr.NewUnprocessable(fmt.Sprintf("%d년 %d월 회계기간이 마감(LOCKED)되어 신규 전표를 작성할 수 없습니다. 전기오류수정전표를 이용하세요.", entryDate.Year(), entryDate.Month()))
	}

	if len(lines) < 2 {
		return nil, apperr.NewBadRequest("복식부기 분개는 최소 2개 이상의 차변/대변 라인이 필요합니다.")
	}

	totalDebit := decimal.Zero
	totalCredit := decimal.Zero

	for _, l := range lines {
		if l.Debit.IsNegative() || l.Credit.IsNegative() {
			return nil, apperr.NewBadRequest("차변 및 대변 금액은 음수일 수 없습니다.")
		}
		if _, exists := s.accounts[l.AccountCode]; !exists {
			return nil, apperr.NewBadRequest(fmt.Sprintf("등록되지 않은 계정과목 코드입니다: %s", l.AccountCode))
		}
		totalDebit = totalDebit.Add(l.Debit)
		totalCredit = totalCredit.Add(l.Credit)
	}

	// Invariant 1: Total Debit == Total Credit
	if !totalDebit.Equal(totalCredit) {
		return nil, apperr.NewUnprocessable(fmt.Sprintf("복식부기 대차불일치 오류: 차변 합계(%s)와 대변 합계(%s)가 일치하지 않습니다.", totalDebit.String(), totalCredit.String()))
	}

	if totalDebit.IsZero() {
		return nil, apperr.NewBadRequest("분개 총액은 0보다 커야 합니다.")
	}

	if sourceLevel == "" {
		sourceLevel = "LEVEL_1_MANUAL"
	}

	now := time.Now().UTC()
	entry := &JournalEntry{
		ID:          uuid.New(),
		StoreID:     storeID,
		EntryNumber: fmt.Sprintf("JRN-%d-%s", now.UnixNano(), uuid.New().String()[:6]),
		EntryDate:   entryDate,
		Description: description,
		Lines:       lines,
		SourceLevel: sourceLevel,
		IsReversed:  false,
		CreatedAt:   now,
	}

	s.entries[entry.ID] = entry
	s.byStore[storeID] = append(s.byStore[storeID], entry)
	return entry, nil
}

// PostManualExpense handles Level 1 manual expense entry
func (s *serviceImpl) PostManualExpense(ctx context.Context, storeID uuid.UUID, input ManualExpenseInput) (*JournalEntry, error) {
	if input.SupplyAmount.IsNegative() || input.VatAmount.IsNegative() {
		return nil, apperr.NewBadRequest("금액은 0 이상이어야 합니다.")
	}
	total := input.SupplyAmount.Add(input.VatAmount)
	if total.IsZero() {
		return nil, apperr.NewBadRequest("비용 금액은 0원보다 커야 합니다.")
	}

	// Determine credit payment account
	var creditAccount string
	switch input.PaymentMethod {
	case "BANK":
		creditAccount = "1010" // 보통예금
	case "CARD":
		creditAccount = "2030" // 미지급금(카드)
	case "CASH":
		creditAccount = "1040" // 현금시재
	case "PAYABLE":
		creditAccount = "2010" // 외상매입금
	default:
		creditAccount = "1010"
	}

	expAcc := input.AccountCode
	if expAcc == "" {
		expAcc = "5010" // Default 식자재
	}

	lines := []JournalLine{
		{AccountCode: expAcc, Debit: input.SupplyAmount, Credit: decimal.Zero, Description: fmt.Sprintf("%s 공급가액 (%s)", input.Description, input.VendorName)},
	}

	if !input.VatAmount.IsZero() {
		// 부가세대급금 (자산 계정 or 매입 부가세 1130) -> 없으면 부가세예수금 차감 or 비용 산입
		// 한국 실무 소상공인: 1060 선급금 or 5010에 포함. 여기서는 2020(부가세예수금) 차변으로 상계
		lines = append(lines, JournalLine{
			AccountCode: "2020",
			Debit:       input.VatAmount,
			Credit:      decimal.Zero,
			Description: fmt.Sprintf("%s 매입세액 공제", input.Description),
		})
	}

	lines = append(lines, JournalLine{
		AccountCode: creditAccount,
		Debit:       decimal.Zero,
		Credit:      total,
		Description: fmt.Sprintf("%s 대금 지급 (%s)", input.Description, input.PaymentMethod),
	})

	return s.PostJournalEntry(ctx, storeID, input.ExpenseDate, fmt.Sprintf("[수기비용] %s - %s", input.VendorName, input.Description), "LEVEL_1_MANUAL", lines)
}

// PostManualRevenue handles Level 1 manual revenue entry
func (s *serviceImpl) PostManualRevenue(ctx context.Context, storeID uuid.UUID, input ManualRevenueInput) (*JournalEntry, error) {
	if input.SupplyAmount.IsNegative() || input.VatAmount.IsNegative() {
		return nil, apperr.NewBadRequest("금액은 0 이상이어야 합니다.")
	}
	total := input.SupplyAmount.Add(input.VatAmount)
	if total.IsZero() {
		return nil, apperr.NewBadRequest("매출 금액은 0원보다 커야 합니다.")
	}

	var revAcc string
	switch input.Channel {
	case "DELIVERY":
		revAcc = "4020" // 배달매출
	case "TAKEOUT":
		revAcc = "4030" // 포장매출
	default:
		revAcc = "4010" // 상품/음식매출
	}

	var debitAcc string
	switch input.PaymentMethod {
	case "CASH":
		debitAcc = "1040" // 현금시재
	case "BANK":
		debitAcc = "1010" // 보통예금
	default:
		debitAcc = "1020" // 카드미수금
	}

	lines := []JournalLine{
		{AccountCode: debitAcc, Debit: total, Credit: decimal.Zero, Description: fmt.Sprintf("%s 수취", input.Description)},
		{AccountCode: revAcc, Debit: decimal.Zero, Credit: input.SupplyAmount, Description: fmt.Sprintf("%s 매출공급가액", input.Description)},
		{AccountCode: "2020", Debit: decimal.Zero, Credit: input.VatAmount, Description: fmt.Sprintf("%s 매출부가세예수금", input.Description)},
	}

	return s.PostJournalEntry(ctx, storeID, input.RevenueDate, fmt.Sprintf("[수기매출] %s", input.Description), "LEVEL_1_MANUAL", lines)
}

// ReverseJournalEntry implements immutable correction via opposite entry
func (s *serviceImpl) ReverseJournalEntry(ctx context.Context, storeID, entryID uuid.UUID, reason string) (*JournalEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	original, exists := s.entries[entryID]
	if !exists || original.StoreID != storeID {
		return nil, apperr.NewNotFound("분개 전표")
	}

	if original.IsReversed {
		return nil, apperr.NewConflict("이미 역분개 정정된 전표입니다.")
	}

	now := time.Now().UTC()
	// If original entry was in a locked period, reversal must be posted in current open period!
	reversalDate := original.EntryDate
	if s.isPeriodLocked(storeID, original.EntryDate) {
		reversalDate = now
	}

	var reversalLines []JournalLine
	for _, l := range original.Lines {
		reversalLines = append(reversalLines, JournalLine{
			AccountCode: l.AccountCode,
			Debit:       l.Credit, // Flip debit and credit
			Credit:      l.Debit,
			Description: fmt.Sprintf("[역분개 정정] %s (%s)", l.Description, reason),
		})
	}

	reversal := &JournalEntry{
		ID:          uuid.New(),
		StoreID:     storeID,
		EntryNumber: fmt.Sprintf("REV-%d-%s", now.UnixNano(), uuid.New().String()[:6]),
		EntryDate:   reversalDate,
		Description: fmt.Sprintf("[역분개 정정: %s] %s", original.EntryNumber, reason),
		Lines:       reversalLines,
		SourceLevel: original.SourceLevel,
		IsReversed:  false,
		CreatedAt:   now,
	}

	original.IsReversed = true
	s.entries[reversal.ID] = reversal
	s.byStore[storeID] = append(s.byStore[storeID], reversal)

	return reversal, nil
}

// PostPriorPeriodAdjustment posts an adjustment entry for past locked period into current open period
func (s *serviceImpl) PostPriorPeriodAdjustment(ctx context.Context, storeID uuid.UUID, originalPeriodYear, originalPeriodMonth int, reason string, lines []JournalLine) (*JournalEntry, error) {
	now := time.Now().UTC()
	description := fmt.Sprintf("[전기오류수정] %d년 %d월 마감기 오차 조정: %s", originalPeriodYear, originalPeriodMonth, reason)
	return s.PostJournalEntry(ctx, storeID, now, description, "LEVEL_1_MANUAL", lines)
}

// ClosePeriod locks accounting period
func (s *serviceImpl) ClosePeriod(ctx context.Context, storeID uuid.UUID, year, month int, closedBy uuid.UUID) (*AccountingPeriod, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if month < 1 || month > 12 {
		return nil, apperr.NewBadRequest("월은 1부터 12 사이여야 합니다.")
	}

	key := fmt.Sprintf("%s-%04d-%02d", storeID.String(), year, month)
	now := time.Now().UTC()

	p := &AccountingPeriod{
		ID:        uuid.New(),
		StoreID:   storeID,
		Year:      year,
		Month:     month,
		Status:    PeriodLocked,
		ClosedAt:  &now,
		ClosedBy:  &closedBy,
		CreatedAt: now,
	}

	s.periods[key] = p
	return p, nil
}

func (s *serviceImpl) GetPeriod(ctx context.Context, storeID uuid.UUID, year, month int) (*AccountingPeriod, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%s-%04d-%02d", storeID.String(), year, month)
	if p, exists := s.periods[key]; exists {
		return p, nil
	}

	// Default open
	return &AccountingPeriod{
		ID:        uuid.Nil,
		StoreID:   storeID,
		Year:      year,
		Month:     month,
		Status:    PeriodOpen,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (s *serviceImpl) CalculatePnL(ctx context.Context, storeID uuid.UUID, fromDate, toDate time.Time) (*ProfitAndLossReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := s.byStore[storeID]

	salesSupply := decimal.Zero
	outputVAT := decimal.Zero
	cogs := decimal.Zero
	platformFees := decimal.Zero
	fixedExpenses := decimal.Zero

	for _, entry := range entries {
		if !fromDate.IsZero() && entry.EntryDate.Before(fromDate) {
			continue
		}
		if !toDate.IsZero() && entry.EntryDate.After(toDate) {
			continue
		}

		for _, line := range entry.Lines {
			switch line.AccountCode {
			case "4010", "4020", "4030", "4040": // Sales revenue supply value (Credit)
				salesSupply = salesSupply.Add(line.Credit).Sub(line.Debit)
			case "2020": // Output VAT (Credit)
				outputVAT = outputVAT.Add(line.Credit).Sub(line.Debit)
			case "5010": // COGS (Debit)
				cogs = cogs.Add(line.Debit).Sub(line.Credit)
			case "5020", "5030": // Fees (Debit)
				platformFees = platformFees.Add(line.Debit).Sub(line.Credit)
			case "5040", "5050", "5055", "5060", "5070", "5080", "5090", "5100", "5110", "5120", "5130": // Fixed/Variable expenses (Debit)
				fixedExpenses = fixedExpenses.Add(line.Debit).Sub(line.Credit)
			}
		}
	}

	grossSales := salesSupply.Add(outputVAT)
	netSales := salesSupply
	grossProfit := netSales.Sub(cogs)
	contribMargin := grossProfit.Sub(platformFees)
	opProfit := contribMargin.Sub(fixedExpenses)

	opRate := decimal.Zero
	if !netSales.IsZero() {
		opRate = opProfit.Div(netSales).Mul(decimal.NewFromInt(100)).Round(2)
	}

	return &ProfitAndLossReport{
		StoreID:             storeID,
		GrossSales:          grossSales,
		OutputVAT:           outputVAT,
		NetSales:            netSales,
		COGS:                cogs,
		GrossProfit:         grossProfit,
		PlatformFees:        platformFees,
		ContributionMargin:  contribMargin,
		FixedExpenses:       fixedExpenses,
		OperatingProfit:     opProfit,
		OperatingProfitRate: opRate,
	}, nil
}

// GetTrialBalance calculates 합계잔액시산표
func (s *serviceImpl) GetTrialBalance(ctx context.Context, storeID uuid.UUID, asOfDate time.Time) (*TrialBalance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := s.byStore[storeID]
	debitTotals := make(map[string]decimal.Decimal)
	creditTotals := make(map[string]decimal.Decimal)

	for _, entry := range entries {
		if !asOfDate.IsZero() && entry.EntryDate.After(asOfDate) {
			continue
		}
		for _, line := range entry.Lines {
			debitTotals[line.AccountCode] = debitTotals[line.AccountCode].Add(line.Debit)
			creditTotals[line.AccountCode] = creditTotals[line.AccountCode].Add(line.Credit)
		}
	}

	var lines []TrialBalanceLine
	totalDebitBal := decimal.Zero
	totalCreditBal := decimal.Zero

	for _, acc := range StandardCOA {
		dTotal := debitTotals[acc.Code]
		cTotal := creditTotals[acc.Code]

		if dTotal.IsZero() && cTotal.IsZero() {
			continue
		}

		var dBal, cBal decimal.Decimal
		if acc.NormalBal == "DEBIT" {
			net := dTotal.Sub(cTotal)
			if net.IsPositive() {
				dBal = net
			} else {
				cBal = net.Abs()
			}
		} else {
			net := cTotal.Sub(dTotal)
			if net.IsPositive() {
				cBal = net
			} else {
				dBal = net.Abs()
			}
		}

		totalDebitBal = totalDebitBal.Add(dBal)
		totalCreditBal = totalCreditBal.Add(cBal)

		lines = append(lines, TrialBalanceLine{
			AccountCode: acc.Code,
			AccountName: acc.Name,
			AccountType: acc.Type,
			DebitTotal:  dTotal,
			CreditTotal: cTotal,
			DebitBal:    dBal,
			CreditBal:   cBal,
		})
	}

	return &TrialBalance{
		StoreID:        storeID,
		AsOfDate:       asOfDate,
		Lines:          lines,
		TotalDebitBal:  totalDebitBal,
		TotalCreditBal: totalCreditBal,
		IsBalanced:     totalDebitBal.Equal(totalCreditBal),
	}, nil
}

// GetGeneralLedger aggregates chronological transactions for specific account
func (s *serviceImpl) GetGeneralLedger(ctx context.Context, storeID uuid.UUID, accountCode string, fromDate, toDate time.Time) (*GeneralLedgerAccount, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	acc, exists := s.accounts[accountCode]
	if !exists {
		return nil, apperr.NewNotFound("계정과목")
	}

	entries := s.byStore[storeID]
	var items []GLEntryItem
	totDebit := decimal.Zero
	totCredit := decimal.Zero
	runningBal := decimal.Zero

	for _, entry := range entries {
		if !fromDate.IsZero() && entry.EntryDate.Before(fromDate) {
			continue
		}
		if !toDate.IsZero() && entry.EntryDate.After(toDate) {
			continue
		}

		for _, line := range entry.Lines {
			if line.AccountCode == accountCode {
				totDebit = totDebit.Add(line.Debit)
				totCredit = totCredit.Add(line.Credit)

				if acc.NormalBal == "DEBIT" {
					runningBal = runningBal.Add(line.Debit).Sub(line.Credit)
				} else {
					runningBal = runningBal.Add(line.Credit).Sub(line.Debit)
				}

				items = append(items, GLEntryItem{
					EntryID:     entry.ID,
					EntryNumber: entry.EntryNumber,
					Date:        entry.EntryDate,
					Description: line.Description,
					Debit:       line.Debit,
					Credit:      line.Credit,
					RunningBal:  runningBal,
				})
			}
		}
	}

	return &GeneralLedgerAccount{
		AccountCode:  acc.Code,
		AccountName:  acc.Name,
		AccountType:  acc.Type,
		Entries:      items,
		TotalDebit:   totDebit,
		TotalCredit:  totCredit,
		FinalBalance: runningBal,
	}, nil
}

func (s *serviceImpl) ListEntries(ctx context.Context, storeID uuid.UUID) ([]*JournalEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byStore[storeID], nil
}
