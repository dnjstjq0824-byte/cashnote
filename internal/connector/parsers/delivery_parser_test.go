package parsers

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"merchant-os/internal/platform/envelope"
)

func TestDeliverySettlementParser(t *testing.T) {
	csvContent := `정산일자,주문금액,취소금액,중개이용료,배달대행료,결제수수료,할인쿠폰,광고비,수수료부가세
2026-10-25,500000,0,34000,45000,15000,5000,20000,11900
2026-10-26,300000,25000,18700,27000,8250,0,0,5395
`
	storeID := uuid.New()
	records, err := ParseDeliverySettlementCSV(storeID, PlatformBaemin, strings.NewReader(csvContent))
	if err != nil {
		t.Fatalf("failed to parse delivery CSV: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	r1 := records[0]
	if r1.SourceLevel != envelope.Level2File {
		t.Errorf("expected Level 2 provenance, got %s", r1.SourceLevel)
	}

	if !r1.GrossOrderAmount.Equal(decimal.RequireFromString("500000")) {
		t.Errorf("expected gross order 500000, got %s", r1.GrossOrderAmount)
	}

	// 500,000 - (34,000 + 45,000 + 15,000 + 5,000 + 20,000 + 11,900 = 130,900) = 369,100
	expectedNet1 := decimal.RequireFromString("369100")
	if !r1.NetSettledAmount.Equal(expectedNet1) {
		t.Errorf("expected net settled %s, got %s", expectedNet1, r1.NetSettledAmount)
	}

	r2 := records[1]
	// Net = (300,000 - 25,000 = 275,000) - (18,700 + 27,000 + 8,250 + 5,395 = 59,345) = 215,655
	expectedNet2 := decimal.RequireFromString("215655")
	if !r2.NetSettledAmount.Equal(expectedNet2) {
		t.Errorf("expected net settled %s, got %s", expectedNet2, r2.NetSettledAmount)
	}
}

