package envelope

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ProvenanceLevel represents the 4-level data provenance hierarchy
type ProvenanceLevel string

const (
	Level1Manual     ProvenanceLevel = "LEVEL_1_MANUAL"     // Direct user input
	Level2File       ProvenanceLevel = "LEVEL_2_FILE"       // Verified file upload (Excel/CSV)
	Level3API        ProvenanceLevel = "LEVEL_3_API"        // Official partner/authorized API
	Level4Reconciled ProvenanceLevel = "LEVEL_4_RECONCILED" // Multi-way cross-reconciled
)

// DataProvenance represents data origin, verification level, and timestamp
type DataProvenance struct {
	SourceLevel ProvenanceLevel `json:"source_level"`
	Verified    bool            `json:"verified"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Fingerprint string          `json:"fingerprint,omitempty"`
}

// Meta contains pagination and request context metadata
type Meta struct {
	Page       int   `json:"page,omitempty"`
	PerPage    int   `json:"per_page,omitempty"`
	TotalCount int64 `json:"total_count,omitempty"`
	TotalPages int   `json:"total_pages,omitempty"`
}

// Response is the universal HTTP API response envelope across BN Merchant OS
type Response struct {
	RequestID      string          `json:"request_id"`
	Data           any             `json:"data,omitempty"`
	Meta           *Meta           `json:"meta,omitempty"`
	DataProvenance *DataProvenance `json:"data_provenance,omitempty"`
	Error          any             `json:"error,omitempty"`
}

// WriteJSON sends a JSON response with status code and standard envelope
func WriteJSON(w http.ResponseWriter, r *http.Request, status int, data any, meta *Meta, prov *DataProvenance) {
	requestID := r.Header.Get("X-Request-Id")
	if requestID == "" {
		requestID = uuid.NewString()
	}

	resp := Response{
		RequestID:      requestID,
		Data:           data,
		Meta:           meta,
		DataProvenance: prov,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Request-Id", requestID)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

// WriteError sends a standard error response
func WriteError(w http.ResponseWriter, r *http.Request, status int, errPayload any) {
	requestID := r.Header.Get("X-Request-Id")
	if requestID == "" {
		requestID = uuid.NewString()
	}

	resp := Response{
		RequestID: requestID,
		Error:     errPayload,
	}

	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.Header().Set("X-Request-Id", requestID)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

