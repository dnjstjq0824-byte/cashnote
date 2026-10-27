package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"merchant-os/internal/business"
	"merchant-os/internal/connector/parsers"
	"merchant-os/internal/dom_02_arch"
	"merchant-os/internal/identity"
	"merchant-os/internal/ledger"
	"merchant-os/internal/platform/envelope"
	apperr "merchant-os/internal/platform/errors"
	"merchant-os/internal/platform/middleware"
	"merchant-os/internal/store"
)

func setupTestRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestIDMiddleware)
	r.Use(middleware.TenantContextMiddleware)
	r.Use(middleware.IdempotencyMiddleware)
	r.Use(middleware.Recoverer)

	archRepo := dom_02_arch.NewMemoryRepository()
	archSvc := dom_02_arch.NewService(archRepo)
	idSvc := identity.NewService()
	bizSvc := business.NewService()
	storeSvc := store.NewService()
	ledgerSvc := ledger.NewService()

	r.Route("/v1", func(api chi.Router) {
		api.Get("/health/live", dom_02_arch.HandleLiveCheck(archSvc))
		api.Get("/health/ready", dom_02_arch.HandleReadyCheck(archSvc))
		api.Get("/system/capabilities", dom_02_arch.HandleListCapabilities(archSvc))

		api.Post("/auth/register", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Email    string `json:"email"`
				Password string `json:"password"`
				FullName string `json:"full_name"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			user, err := idSvc.Register(r.Context(), req.Email, req.Password, req.FullName)
			if err != nil {
				if appErr, ok := err.(*apperr.AppError); ok {
					envelope.WriteError(w, r, appErr.HTTPStatus, appErr.ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
				return
			}
			envelope.WriteJSON(w, r, http.StatusCreated, user, nil, nil)
		})

		api.Post("/auth/login", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			user, token, err := idSvc.Login(r.Context(), req.Email, req.Password)
			if err != nil {
				envelope.WriteError(w, r, http.StatusUnauthorized, apperr.NewUnauthorized("로그인 실패").ToProblemDetails(r.URL.Path))
				return
			}
			envelope.WriteJSON(w, r, http.StatusOK, map[string]any{"user": user, "token": token}, nil, nil)
		})

		api.Post("/businesses", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				OwnerUserID        uuid.UUID `json:"owner_user_id"`
				RegistrationNumber string    `json:"registration_number"`
				CompanyName        string    `json:"company_name"`
				RepresentativeName string    `json:"representative_name"`
				OpeningDate        string    `json:"opening_date"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			biz, err := bizSvc.RegisterBusiness(r.Context(), req.OwnerUserID, req.RegistrationNumber, req.CompanyName, req.RepresentativeName, req.OpeningDate)
			if err != nil {
				if appErr, ok := err.(*apperr.AppError); ok {
					envelope.WriteError(w, r, appErr.HTTPStatus, appErr.ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
				return
			}
			envelope.WriteJSON(w, r, http.StatusCreated, biz, nil, nil)
		})

		api.Post("/businesses/{businessId}/verify-ownership", func(w http.ResponseWriter, r *http.Request) {
			bizID, _ := uuid.Parse(chi.URLParam(r, "businessId"))
			var req struct {
				AuthCode string `json:"auth_code"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			ok, err := bizSvc.VerifyOwnership1Won(r.Context(), bizID, "004", "110-123", req.AuthCode)
			if err != nil {
				envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("인증 실패").ToProblemDetails(r.URL.Path))
				return
			}
			envelope.WriteJSON(w, r, http.StatusOK, map[string]any{"verified": ok}, nil, nil)
		})

		api.Post("/stores", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				BusinessID  uuid.UUID `json:"business_id"`
				OwnerUserID uuid.UUID `json:"owner_user_id"`
				StoreName   string    `json:"store_name"`
				Category    string    `json:"category"`
				Address     string    `json:"address"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			st, err := storeSvc.CreateStore(r.Context(), req.BusinessID, req.OwnerUserID, req.StoreName, req.Category, req.Address)
			if err != nil {
				envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest(err.Error()).ToProblemDetails(r.URL.Path))
				return
			}
			envelope.WriteJSON(w, r, http.StatusCreated, st, nil, nil)
		})

		api.Route("/stores/{storeId}", func(storeRouter chi.Router) {
			storeRouter.Post("/delivery/settlements/parse-csv", func(w http.ResponseWriter, r *http.Request) {
				storeID, _ := uuid.Parse(chi.URLParam(r, "storeId"))
				records, err := parsers.ParseDeliverySettlementCSV(storeID, parsers.PlatformBaemin, r.Body)
				if err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest(err.Error()).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusOK, records, &envelope.Meta{TotalCount: int64(len(records))}, nil)
			})

			storeRouter.Get("/analytics/pnl", func(w http.ResponseWriter, r *http.Request) {
				storeID, _ := uuid.Parse(chi.URLParam(r, "storeId"))
				now := time.Now()
				pnl, _ := ledgerSvc.CalculatePnL(r.Context(), storeID, now.AddDate(0, -1, 0), now)
				envelope.WriteJSON(w, r, http.StatusOK, pnl, nil, nil)
			})
		})
	})

	return r
}

func TestE2E_APIContract(t *testing.T) {
	router := setupTestRouter()

	t.Run("Health and Capabilities endpoints", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/health/live", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var resp envelope.Response
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp.RequestID == "" {
			t.Errorf("expected non-empty request_id")
		}
	})

	t.Run("User Onboarding and Store Lifecycle Flow", func(t *testing.T) {
		// 1. Register
		regBody := []byte(`{"email":"merchant@bn.internal","password":"SecurePassword123!","full_name":"김대표"}`)
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader(regBody))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("registration failed: %d body: %s", rec.Code, rec.Body.String())
		}

		var regResp envelope.Response
		_ = json.Unmarshal(rec.Body.Bytes(), &regResp)
		dataMap := regResp.Data.(map[string]any)
		userID := uuid.MustParse(dataMap["id"].(string))

		// 2. Login
		loginBody := []byte(`{"email":"merchant@bn.internal","password":"SecurePassword123!"}`)
		req = httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(loginBody))
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("login failed: %d", rec.Code)
		}

		// 3. Register Business
		bizJSON := fmt.Sprintf(`{"owner_user_id":"%s","registration_number":"123-45-67890","company_name":"BN카페","representative_name":"김대표","opening_date":"20240101"}`, userID)
		req = httptest.NewRequest(http.MethodPost, "/v1/businesses", bytes.NewReader([]byte(bizJSON)))
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("business registration failed: %d body: %s", rec.Code, rec.Body.String())
		}

		var bizResp envelope.Response
		_ = json.Unmarshal(rec.Body.Bytes(), &bizResp)
		bizMap := bizResp.Data.(map[string]any)
		bizID := uuid.MustParse(bizMap["id"].(string))

		// 4. Verify Ownership via 1-Won Auth Code
		verifyJSON := `{"auth_code":"7788"}`
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/businesses/%s/verify-ownership", bizID), bytes.NewReader([]byte(verifyJSON)))
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("1-won verification failed: %d body: %s", rec.Code, rec.Body.String())
		}

		// 5. Create Store
		storeJSON := fmt.Sprintf(`{"business_id":"%s","owner_user_id":"%s","store_name":"BN카페 역삼점","category":"카페","address":"서울 강남구 역삼동 10"}`, bizID, userID)
		req = httptest.NewRequest(http.MethodPost, "/v1/stores", bytes.NewReader([]byte(storeJSON)))
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("store creation failed: %d body: %s", rec.Code, rec.Body.String())
		}

		var storeResp envelope.Response
		_ = json.Unmarshal(rec.Body.Bytes(), &storeResp)
		stMap := storeResp.Data.(map[string]any)
		storeID := uuid.MustParse(stMap["id"].(string))

		// 6. Level 2 Delivery File Upload Parsing
		csvData := "정산일자,주문금액,취소금액,중개이용료,배달대행료,결제수수료,할인쿠폰,광고비,수수료부가세\n2026-10-28,200000,0,13600,18000,6000,0,10000,4760\n"
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/stores/%s/delivery/settlements/parse-csv", storeID), bytes.NewReader([]byte(csvData)))
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("delivery CSV parse failed: %d body: %s", rec.Code, rec.Body.String())
		}

		// 7. Store P&L Check
		req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/stores/%s/analytics/pnl", storeID), nil)
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("pnl query failed: %d body: %s", rec.Code, rec.Body.String())
		}
	})
}
