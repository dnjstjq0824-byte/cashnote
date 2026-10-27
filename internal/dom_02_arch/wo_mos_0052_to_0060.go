package dom_02_arch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"merchant-os/internal/platform/envelope"
	apperr "merchant-os/internal/platform/errors"
)

// MonorepoManifest represents the Monorepo directory verification for WO-MOS-0052
type MonorepoManifest struct {
	Apps     []string `json:"apps"`
	Packages []string `json:"packages"`
	Services []string `json:"services"`
	Internal []string `json:"internal"`
}

// GetMonorepoManifest returns the official modular monorepo structure
func GetMonorepoManifest() MonorepoManifest {
	return MonorepoManifest{
		Apps:     []string{"apps/web-merchant", "apps/web-admin", "apps/mobile"},
		Packages: []string{"packages/ui-design-system", "packages/api-client"},
		Services: []string{"cmd/api-server", "cmd/connector-worker"},
		Internal: []string{
			"internal/platform",
			"internal/identity",
			"internal/business",
			"internal/store",
			"internal/revenue",
			"internal/settlement",
			"internal/reconciliation",
			"internal/ledger",
			"internal/expense",
			"internal/tax",
			"internal/pos",
			"internal/delivery",
			"internal/crm",
			"internal/reviews",
			"internal/ai",
			"internal/commerce",
			"internal/community",
			"internal/admin",
			"internal/dom_02_arch",
		},
	}
}

// ComputeTransactionFingerprint generates SHA-256 fingerprint for idempotency (WO-MOS-0053/0059)
func ComputeTransactionFingerprint(storeID uuid.UUID, provider string, originalID string, amount decimal.Decimal) string {
	raw := fmt.Sprintf("%s:%s:%s:%s", storeID.String(), strings.ToUpper(provider), originalID, amount.StringFixed(4))
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// HandleCalculateSettlement computes settlement amounts with precision (WO-MOS-0053)
func HandleCalculateSettlement(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		salesStr := r.URL.Query().Get("amount")
		feeRateStr := r.URL.Query().Get("fee_rate")

		salesAmount, err := decimal.NewFromString(salesStr)
		if err != nil {
			errPayload := apperr.NewBadRequest("유효한 매출 금액(amount) 파라미터를 입력하세요.").ToProblemDetails(r.URL.Path)
			envelope.WriteError(w, r, http.StatusBadRequest, errPayload)
			return
		}

		feeRate, err := decimal.NewFromString(feeRateStr)
		if err != nil {
			feeRate = decimal.NewFromFloat(0.015) // Default 1.5% fee
		}

		fee, netPayout, err := svc.CalculateSampleFinancials(r.Context(), salesAmount, feeRate)
		if err != nil {
			if appErr, ok := err.(*apperr.AppError); ok {
				envelope.WriteError(w, r, appErr.HTTPStatus, appErr.ToProblemDetails(r.URL.Path))
				return
			}
			envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
			return
		}

		result := map[string]any{
			"sales_amount": salesAmount.StringFixed(4),
			"fee_rate":     feeRate.StringFixed(4),
			"fee_amount":   fee.StringFixed(4),
			"net_payout":   netPayout.StringFixed(4),
		}

		envelope.WriteJSON(w, r, http.StatusOK, result, nil, &envelope.DataProvenance{
			SourceLevel: envelope.Level3API,
			Verified:    true,
			UpdatedAt:   svc.GetLiveness(context.Background()).Timestamp,
		})
	}
}
