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

// Account represents a Chart of Accounts entry
type Account struct {
	Code      string      `json:"code"`
	Name      string      `json:"name"`
	Type      AccountType `json:"type"`
	NormalBal string      `json:"normal_balance"` // DEBIT or CREDIT
}

// Standard Chart of Accounts for Korean small businesses
var StandardCOA = []Account{
	{Code: "1010", Name: "보통예금(은행계좌)", Type: AccountAsset, NormalBal: "DEBIT"},
	{Code: "1020", Name: "카드미수금(매입완료/입금예정)", Type: AccountAsset, NormalBal: "DEBIT"},
	{Code: "1030", Name: "배달앱미수금(정산예정)", Type: AccountAsset, NormalBal: "DEBIT"},
	{Code: "1040", Name: "현금시재", Type: AccountAsset, NormalBal: "DEBIT"},
	{Code: "2010", Name: "외상매입금(식자재/비품미지급)", Type: AccountLiability, NormalBal: "CREDIT"},
	{Code: "2020", Name: "부가세예수금(매출부가세)", Type: AccountLiability, NormalBal: "CREDIT"},
	{Code: "4010", Name: "상품/음식매출", Type: AccountRevenue, NormalBal: "CREDIT"},
	{Code: "4020", Name: "배달매출", Type: AccountRevenue, NormalBal: "CREDIT"},
	{Code: "5010", Name: "식자재/원재료매입(매출원가)", Type: AccountExpense, NormalBal: "DEBIT"},
	{Code: "5020", Name: "카드결제/PG수수료", Type: AccountExpense, NormalBal: "DEBIT"},
	{Code: "5030", Name: "배달중개수수료및배달비", Type: AccountExpense, NormalBal: "DEBIT"},
	{Code: "5040", Name: "임차료및관리비", Type: AccountExpense, NormalBal: "DEBIT"},
	{Code: "5050", Name: "인건비및급여", Type: AccountExpense, NormalBal: "DEBIT"},
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
	IsReversed  bool          `json:"is_reversed"`
	CreatedAt   time.Time     `json:"created_at"`
}

// ProfitAndLossReport represents Korean Small Business P&L statement
type ProfitAndLossReport struct {
	StoreID             uuid.UUID       `json:"store_id"`
	GrossSales          decimal.Decimal `json:"gross_sales"`           // 총매출액
	OutputVAT           decimal.Decimal `json:"output_vat"`            // 부가세예수금
	NetSales            decimal.Decimal `json:"net_sales"`             // 순매출액
	COGS                decimal.Decimal `json:"cogs"`                  // 매출원가(식자재)
	GrossProfit         decimal.Decimal `json:"gross_profit"`          // 매출총이익
	PlatformFees        decimal.Decimal `json:"platform_fees"`         // 결제/배달수수료
	ContributionMargin  decimal.Decimal `json:"contribution_margin"`   // 공헌이익
	FixedExpenses       decimal.Decimal `json:"fixed_expenses"`        // 임차료/인건비 등
	OperatingProfit     decimal.Decimal `json:"operating_profit"`      // 추정영업이익
	OperatingProfitRate decimal.Decimal `json:"operating_profit_rate"` // 영업이익률 (%)
}

// Service defines double-entry ledger operations
type Service interface {
	PostJournalEntry(ctx context.Context, storeID uuid.UUID, entryDate time.Time, description string, lines []JournalLine) (*JournalEntry, error)
	ReverseJournalEntry(ctx context.Context, storeID, entryID uuid.UUID, reason string) (*JournalEntry, error)
	CalculatePnL(ctx context.Context, storeID uuid.UUID, fromDate, toDate time.Time) (*ProfitAndLossReport, error)
	ListEntries(ctx context.Context, storeID uuid.UUID) ([]*JournalEntry, error)
}

type serviceImpl struct {
	mu      sync.RWMutex
	entries map[uuid.UUID]*JournalEntry
	byStore map[uuid.UUID][]*JournalEntry
}

func NewService() Service {
	return &serviceImpl{
		entries: make(map[uuid.UUID]*JournalEntry),
		byStore: make(map[uuid.UUID][]*JournalEntry),
	}
}

func (s *serviceImpl) PostJournalEntry(ctx context.Context, storeID uuid.UUID, entryDate time.Time, description string, lines []JournalLine) (*JournalEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(lines) < 2 {
		return nil, apperr.NewBadRequest("복식부기 분개는 최소 2개 이상의 차변/대변 라인이 필요합니다.")
	}

	totalDebit := decimal.Zero
	totalCredit := decimal.Zero

	for _, l := range lines {
		if l.Debit.IsNegative() || l.Credit.IsNegative() {
			return nil, apperr.NewBadRequest("차변 및 대변 금액은 음수일 수 없습니다.")
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

	now := time.Now().UTC()
	entry := &JournalEntry{
		ID:          uuid.New(),
		StoreID:     storeID,
		EntryNumber: fmt.Sprintf("JRN-%d-%s", now.UnixNano(), uuid.New().String()[:6]),
		EntryDate:   entryDate,
		Description: description,
		Lines:       lines,
		IsReversed:  false,
		CreatedAt:   now,
	}

	s.entries[entry.ID] = entry
	s.byStore[storeID] = append(s.byStore[storeID], entry)
	return entry, nil
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

	var reversalLines []JournalLine
	for _, l := range original.Lines {
		reversalLines = append(reversalLines, JournalLine{
			AccountCode: l.AccountCode,
			Debit:       l.Credit, // Flip debit and credit
			Credit:      l.Debit,
			Description: fmt.Sprintf("[역분개 정정] %s (%s)", l.Description, reason),
		})
	}

	now := time.Now().UTC()
	reversal := &JournalEntry{
		ID:          uuid.New(),
		StoreID:     storeID,
		EntryNumber: fmt.Sprintf("REV-%d-%s", now.UnixNano(), uuid.New().String()[:6]),
		EntryDate:   now,
		Description: fmt.Sprintf("[역분개 정정: %s] %s", original.EntryNumber, reason),
		Lines:       reversalLines,
		IsReversed:  false,
		CreatedAt:   now,
	}

	original.IsReversed = true
	s.entries[reversal.ID] = reversal
	s.byStore[storeID] = append(s.byStore[storeID], reversal)

	return reversal, nil
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
		for _, line := range entry.Lines {
			switch line.AccountCode {
			case "4010", "4020": // Sales revenue supply value (Credit)
				salesSupply = salesSupply.Add(line.Credit).Sub(line.Debit)
			case "2020": // Output VAT (Credit)
				outputVAT = outputVAT.Add(line.Credit).Sub(line.Debit)
			case "5010": // COGS (Debit)
				cogs = cogs.Add(line.Debit).Sub(line.Credit)
			case "5020", "5030": // Fees (Debit)
				platformFees = platformFees.Add(line.Debit).Sub(line.Credit)
			case "5040", "5050": // Fixed expenses (Debit)
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

func (s *serviceImpl) ListEntries(ctx context.Context, storeID uuid.UUID) ([]*JournalEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byStore[storeID], nil
}
