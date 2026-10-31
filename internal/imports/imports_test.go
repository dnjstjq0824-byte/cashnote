package imports

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"merchant-os/internal/platform/envelope"
)

func TestImportsService(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	storeID := uuid.New()

	t.Run("Upload and parse delivery CSV file as Level 2", func(t *testing.T) {
		csvContent := "정산일자,주문금액,취소금액,중개이용료,배달대행료,결제수수료,할인쿠폰,광고비,수수료부가세\n2026-10-30,100000,0,6800,9000,3000,0,0,1880\n"

		res, err := svc.UploadAndParse(ctx, storeID, TypeDeliveryBaemin, "baemin_settlement_202610.csv", strings.NewReader(csvContent))
		if err != nil {
			t.Fatalf("failed to upload and parse: %v", err)
		}

		if res.ParsedRowCount != 1 {
			t.Errorf("expected 1 row parsed, got %d", res.ParsedRowCount)
		}
		if res.SourceLevel != envelope.Level2File {
			t.Errorf("expected Level 2 file provenance, got %s", res.SourceLevel)
		}
		if !res.TotalGrossAmount.Equal(decimal.RequireFromString("100000")) {
			t.Errorf("expected 100,000 gross, got %s", res.TotalGrossAmount)
		}
		// 100,000 - (6800 + 9000 + 3000 + 1880 = 20,680) = 79,320
		expectedNet := decimal.RequireFromString("79320")
		if !res.TotalNetAmount.Equal(expectedNet) {
			t.Errorf("expected %s net, got %s", expectedNet, res.TotalNetAmount)
		}
	})
}

