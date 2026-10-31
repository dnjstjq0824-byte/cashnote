package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	"merchant-os/internal/platform/envelope"
	apperr "merchant-os/internal/platform/errors"
)

type contextKey string

const (
	StoreIdKey    contextKey = "store_id"
	BusinessIdKey contextKey = "business_id"
	UserIdKey     contextKey = "user_id"
	RequestIdKey  contextKey = "request_id"
)

// RequestIDMiddleware ensures each request has an X-Request-Id header and context value
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-Id")
		if reqID == "" {
			reqID = uuid.NewString()
		}
		w.Header().Set("X-Request-Id", reqID)
		ctx := context.WithValue(r.Context(), RequestIdKey, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// TenantContextMiddleware extracts X-Store-Id and X-Business-Id from headers and populates context
func TenantContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		storeIDStr := r.Header.Get("X-Store-Id")
		if storeIDStr != "" {
			if parsedStore, err := uuid.Parse(storeIDStr); err == nil {
				ctx = context.WithValue(ctx, StoreIdKey, parsedStore)
			}
		}

		bizIDStr := r.Header.Get("X-Business-Id")
		if bizIDStr != "" {
			if parsedBiz, err := uuid.Parse(bizIDStr); err == nil {
				ctx = context.WithValue(ctx, BusinessIdKey, parsedBiz)
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// MemoryIdempotencyStore provides in-memory deduplication for state-changing requests
type MemoryIdempotencyStore struct {
	mu   sync.RWMutex
	seen map[string]time.Time
}

var globalIdempotencyStore = &MemoryIdempotencyStore{
	seen: make(map[string]time.Time),
}

// IdempotencyMiddleware ensures state-changing POST/PUT requests with Idempotency-Key are deduplicated
func IdempotencyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			key := r.Header.Get("Idempotency-Key")
			if key != "" {
				globalIdempotencyStore.mu.Lock()
				if lastSeen, exists := globalIdempotencyStore.seen[key]; exists && time.Since(lastSeen) < 10*time.Minute {
					globalIdempotencyStore.mu.Unlock()
					problem := apperr.ProblemDetails{
						Type:   "https://api.merchant-os.bookingnetwork.internal/errors/" + apperr.CodeIdempotencyConflict,
						Title:  "동일한 멱등성 키로 이미 처리 중이거나 완료된 요청입니다.",
						Status: http.StatusConflict,
						Detail: "요청이 이미 수신되어 처리 중입니다. 중복 결제/전송을 방지합니다.",
						Code:   apperr.CodeIdempotencyConflict,
					}
					envelope.WriteError(w, r, http.StatusConflict, problem)
					return
				}
				globalIdempotencyStore.seen[key] = time.Now()
				globalIdempotencyStore.mu.Unlock()
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Recoverer catches panics and formats them as RFC 9457 Internal Server Error JSON
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				appErr := apperr.NewInternal(nil)
				problem := appErr.ToProblemDetails(r.URL.Path)
				envelope.WriteError(w, r, http.StatusInternalServerError, problem)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// GetStoreID returns the Store UUID from context if present
func GetStoreID(ctx context.Context) (uuid.UUID, bool) {
	val, ok := ctx.Value(StoreIdKey).(uuid.UUID)
	return val, ok
}

// GetBusinessID returns the Business UUID from context if present
func GetBusinessID(ctx context.Context) (uuid.UUID, bool) {
	val, ok := ctx.Value(BusinessIdKey).(uuid.UUID)
	return val, ok
}

