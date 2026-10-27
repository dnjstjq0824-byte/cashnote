package dom_02_arch

import (
	"context"
	"net/http"

	"merchant-os/internal/platform/envelope"
)

// HandleLiveCheck returns 200 OK liveness check
func HandleLiveCheck(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := svc.GetLiveness(r.Context())
		envelope.WriteJSON(w, r, http.StatusOK, status, nil, &envelope.DataProvenance{
			SourceLevel: envelope.Level3API,
			Verified:    true,
			UpdatedAt:   status.Timestamp,
		})
	}
}

// HandleReadyCheck returns 200 OK readiness check
func HandleReadyCheck(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := svc.GetReadiness(r.Context())
		envelope.WriteJSON(w, r, http.StatusOK, status, nil, &envelope.DataProvenance{
			SourceLevel: envelope.Level3API,
			Verified:    true,
			UpdatedAt:   status.Timestamp,
		})
	}
}

// HandleListCapabilities returns system API integration readiness and regulatory gates
func HandleListCapabilities(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caps := svc.ListSystemCapabilities(r.Context())
		envelope.WriteJSON(w, r, http.StatusOK, caps, &envelope.Meta{
			TotalCount: int64(len(caps)),
		}, &envelope.DataProvenance{
			SourceLevel: envelope.Level3API,
			Verified:    true,
			UpdatedAt:   svc.GetLiveness(context.Background()).Timestamp,
		})
	}
}
