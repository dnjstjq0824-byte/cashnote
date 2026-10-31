package dom_02_arch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"merchant-os/internal/platform/envelope"
	apperr "merchant-os/internal/platform/errors"
)

func TestWO_MOS_0052_MonorepoManifest(t *testing.T) {
	manifest := GetMonorepoManifest()
	if len(manifest.Apps) < 3 {
		t.Errorf("expected at least 3 apps (web-merchant, web-admin, mobile), got %d", len(manifest.Apps))
	}
	if len(manifest.Internal) < 15 {
		t.Errorf("expected at least 15 domain packages, got %d", len(manifest.Internal))
	}
}

func TestWO_MOS_0053_DecimalPrecision(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo)

	salesAmount := decimal.RequireFromString("100000.0000")
	feeRate := decimal.RequireFromString("0.0150") // 1.5%

	fee, netPayout, err := svc.CalculateSampleFinancials(context.Background(), salesAmount, feeRate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedFee := decimal.RequireFromString("1500")
	expectedPayout := decimal.RequireFromString("98500")

	if !fee.Equal(expectedFee) {
		t.Errorf("expected fee %s, got %s", expectedFee, fee)
	}
	if !netPayout.Equal(expectedPayout) {
		t.Errorf("expected net payout %s, got %s", expectedPayout, netPayout)
	}
	// Invariant: fee + netPayout == salesAmount
	if !fee.Add(netPayout).Equal(salesAmount) {
		t.Errorf("financial invariant violated: fee + netPayout != salesAmount")
	}
}

func TestWO_MOS_0059_IdempotencyAndFingerprint(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo)

	storeID := uuid.New()
	amount := decimal.RequireFromString("55000.0000")
	fp := ComputeTransactionFingerprint(storeID, "SHINHAN", "AUTH-9912", amount)

	if fp == "" {
		t.Fatalf("expected non-empty fingerprint")
	}

	// First time should pass
	if err := svc.VerifyIdempotency(context.Background(), storeID, fp); err != nil {
		t.Fatalf("first submission must pass: %v", err)
	}

	// Second time with identical fingerprint must return Conflict (409)
	err := svc.VerifyIdempotency(context.Background(), storeID, fp)
	if err == nil {
		t.Fatalf("duplicate fingerprint submission must fail with conflict")
	}

	appErr, ok := err.(*apperr.AppError)
	if !ok || appErr.HTTPStatus != http.StatusConflict {
		t.Errorf("expected 409 Conflict, got: %v", err)
	}
}

func TestWO_MOS_0060_RFC9457ErrorMapping(t *testing.T) {
	t.Run("Blocked Connector produces RFC 9457 problem details", func(t *testing.T) {
		err := apperr.NewBlockedConnector("배달의민족", "Level 2 정산 엑셀 업로드로 대체 가능합니다.")
		problem := err.ToProblemDetails("/v1/stores/test/delivery/sync")

		if problem.Status != http.StatusUnprocessableEntity {
			t.Errorf("expected 422, got %d", problem.Status)
		}
		if problem.Code != apperr.CodeConnectorExternalBlocked {
			t.Errorf("expected code CONNECTOR_EXTERNAL_BLOCKED, got %s", problem.Code)
		}
		if problem.Title == "" || problem.Detail == "" {
			t.Errorf("expected non-empty localized Korean message")
		}
	})

	t.Run("HTTP handler error envelope formatting", func(t *testing.T) {
		repo := NewMemoryRepository()
		svc := NewService(repo)

		req := httptest.NewRequest(http.MethodGet, "/v1/test/calc?amount=-100", nil)
		rec := httptest.NewRecorder()

		handler := HandleCalculateSettlement(svc)
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", rec.Code)
		}

		var resp envelope.Response
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode error envelope: %v", err)
		}
		if resp.Error == nil {
			t.Fatalf("expected error field in envelope")
		}
	})
}

