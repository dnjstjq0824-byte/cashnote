package ai

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestAiService(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	storeID := uuid.New()

	t.Run("Insufficient historical data (<14 days) is rejected", func(t *testing.T) {
		_, err := svc.GenerateForecast(ctx, storeID, 10, decimal.RequireFromString("1000000"))
		if err == nil {
			t.Fatalf("expected error when historical data is less than 14 days")
		}
	})

	t.Run("Valid data generates 7-day forecast with confidence intervals", func(t *testing.T) {
		report, err := svc.GenerateForecast(ctx, storeID, 30, decimal.RequireFromString("1000000"))
		if err != nil {
			t.Fatalf("failed to generate forecast: %v", err)
		}

		if len(report.Next7Days) != 7 {
			t.Errorf("expected 7 forecast days, got %d", len(report.Next7Days))
		}

		if report.TotalPredictedRevenue.IsZero() {
			t.Errorf("expected non-zero total revenue")
		}

		f1 := report.Next7Days[0]
		if f1.LowerBound80.GreaterThan(f1.PredictedAmount) || f1.UpperBound80.LessThan(f1.PredictedAmount) {
			t.Errorf("80%% confidence interval invalid")
		}
		if f1.LowerBound95.GreaterThan(f1.LowerBound80) || f1.UpperBound95.LessThan(f1.UpperBound80) {
			t.Errorf("95%% confidence interval must enclose 80%% interval")
		}
	})
}

