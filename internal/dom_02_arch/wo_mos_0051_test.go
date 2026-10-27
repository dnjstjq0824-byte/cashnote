package dom_02_arch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"merchant-os/internal/platform/envelope"
)

func TestWO_MOS_0051_HealthChecks(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo)

	t.Run("GET /v1/health/live returns UP and valid envelope", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/health/live", nil)
		rec := httptest.NewRecorder()

		handler := HandleLiveCheck(svc)
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		var resp envelope.Response
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse envelope response: %v", err)
		}

		if resp.RequestID == "" {
			t.Errorf("expected non-empty request_id in envelope")
		}

		if resp.DataProvenance == nil || resp.DataProvenance.SourceLevel != envelope.Level3API {
			t.Errorf("expected DataProvenance with LEVEL_3_API")
		}
	})

	t.Run("GET /v1/health/ready returns READY and valid checks", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/health/ready", nil)
		rec := httptest.NewRecorder()

		handler := HandleReadyCheck(svc)
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("GET /v1/system/capabilities verifies regulatory and fallback levels", func(t *testing.T) {
		caps := svc.ListSystemCapabilities(context.Background())
		if len(caps) == 0 {
			t.Fatalf("expected non-empty capabilities")
		}

		var blockedDelivery bool
		for _, c := range caps {
			if c.ServiceCode == "API-KR-010" {
				if c.Status != "BLOCKED" || c.FallbackLevel != "LEVEL_2_FILE" {
					t.Errorf("delivery platform API-KR-010 must be BLOCKED with LEVEL_2_FILE fallback, got status=%s, fallback=%s", c.Status, c.FallbackLevel)
				}
				blockedDelivery = true
			}
		}

		if !blockedDelivery {
			t.Errorf("expected API-KR-010 capability to be present in catalog")
		}
	})
}
