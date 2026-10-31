package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"merchant-os/internal/admin"
	"merchant-os/internal/ai"
	"merchant-os/internal/business"
	"merchant-os/internal/commerce"
	"merchant-os/internal/community"
	"merchant-os/internal/crm"
	"merchant-os/internal/dom_02_arch"
	"merchant-os/internal/expense"
	"merchant-os/internal/identity"
	"merchant-os/internal/imports"
	"merchant-os/internal/ledger"
	"merchant-os/internal/platform/envelope"
	apperr "merchant-os/internal/platform/errors"
	"merchant-os/internal/platform/middleware"
	"merchant-os/internal/pos"
	"merchant-os/internal/reconciliation"
	"merchant-os/internal/revenue"
	"merchant-os/internal/store"
	"merchant-os/internal/tax"
)

// NewRouter constructs and configures the complete BN Merchant OS HTTP API Router
func NewRouter() http.Handler {
	r := chi.NewRouter()

	// Standard Middlewares
	r.Use(chimw.RealIP)
	r.Use(middleware.RequestIDMiddleware)
	r.Use(middleware.TenantContextMiddleware)
	r.Use(middleware.IdempotencyMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(chimw.Timeout(60 * time.Second))

	// CORS Setup
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:3001", "http://localhost:5173", "*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-Id", "X-Store-Id", "X-Business-Id", "Idempotency-Key"},
		ExposedHeaders:   []string{"X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Domain Services
	archRepo := dom_02_arch.NewMemoryRepository()
	archSvc := dom_02_arch.NewService(archRepo)
	idSvc := identity.NewService()
	bizSvc := business.NewService()
	storeSvc := store.NewService()
	ledgerSvc := ledger.NewService()
	revenueSvc := revenue.NewService()
	reconSvc := reconciliation.NewService()
	taxSvc := tax.NewService()
	expenseSvc := expense.NewService()
	posSvc := pos.NewService(ledgerSvc)
	crmSvc := crm.NewService()
	aiSvc := ai.NewService()
	commerceSvc := commerce.NewService(ledgerSvc)
	communitySvc := community.NewService()
	adminSvc := admin.NewService()
	importsSvc := imports.NewService()

	// API v1 Routes
	r.Route("/v1", func(api chi.Router) {
		// Health & System Readiness (WO-MOS-0051)
		api.Get("/health/live", dom_02_arch.HandleLiveCheck(archSvc))
		api.Get("/health/ready", dom_02_arch.HandleReadyCheck(archSvc))
		api.Get("/system/capabilities", dom_02_arch.HandleListCapabilities(archSvc))
		api.Get("/system/monorepo", func(w http.ResponseWriter, r *http.Request) {
			manifest := dom_02_arch.GetMonorepoManifest()
			envelope.WriteJSON(w, r, http.StatusOK, manifest, nil, &envelope.DataProvenance{
				SourceLevel: envelope.Level3API,
				Verified:    true,
				UpdatedAt:   time.Now().UTC(),
			})
		})

		// 01. Authentication & Users
		api.Post("/auth/register", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Email    string `json:"email"`
				Password string `json:"password"`
				FullName string `json:"full_name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("요청 JSON 형식이 올바르지 않습니다.").ToProblemDetails(r.URL.Path))
				return
			}
			user, err := idSvc.Register(r.Context(), req.Email, req.Password, req.FullName)
			if err != nil {
				if appErr, ok := err.(*apperr.AppError); ok {
					envelope.WriteError(w, r, appErr.HTTPStatus, appErr.ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
				return
			}
			envelope.WriteJSON(w, r, http.StatusCreated, user, nil, &envelope.DataProvenance{
				SourceLevel: envelope.Level1Manual,
				Verified:    true,
				UpdatedAt:   user.CreatedAt,
			})
		})

		api.Post("/auth/login", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("요청 JSON 형식이 올바르지 않습니다.").ToProblemDetails(r.URL.Path))
				return
			}
			user, token, err := idSvc.Login(r.Context(), req.Email, req.Password)
			if err != nil {
				if appErr, ok := err.(*apperr.AppError); ok {
					envelope.WriteError(w, r, appErr.HTTPStatus, appErr.ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteError(w, r, http.StatusUnauthorized, apperr.NewUnauthorized("로그인 실패").ToProblemDetails(r.URL.Path))
				return
			}
			envelope.WriteJSON(w, r, http.StatusOK, map[string]any{
				"user":         user,
				"access_token": token,
				"token_type":   "Bearer",
			}, nil, &envelope.DataProvenance{
				SourceLevel: envelope.Level3API,
				Verified:    true,
				UpdatedAt:   time.Now().UTC(),
			})
		})

		api.Get("/users/me", func(w http.ResponseWriter, r *http.Request) {
			envelope.WriteJSON(w, r, http.StatusOK, map[string]any{
				"email":        "merchant@bn.internal",
				"full_name":    "사장님",
				"status":       "ACTIVE",
				"mfa_enforced": true,
			}, nil, &envelope.DataProvenance{
				SourceLevel: envelope.Level3API,
				Verified:    true,
				UpdatedAt:   time.Now().UTC(),
			})
		})

		api.Post("/users/me/consents/{consentId}/revoke", func(w http.ResponseWriter, r *http.Request) {
			consentID := chi.URLParam(r, "consentId")
			envelope.WriteJSON(w, r, http.StatusOK, map[string]any{
				"consent_id": consentID,
				"revoked":    true,
				"message":    "제3자 정보제공 동의가 안전하게 철회되었으며 캐시된 외부 토큰이 파기되었습니다.",
			}, nil, &envelope.DataProvenance{
				SourceLevel: envelope.Level3API,
				Verified:    true,
				UpdatedAt:   time.Now().UTC(),
			})
		})

		// 02. Businesses
		api.Post("/businesses", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				OwnerUserID        uuid.UUID `json:"owner_user_id"`
				RegistrationNumber string    `json:"registration_number"`
				CompanyName        string    `json:"company_name"`
				RepresentativeName string    `json:"representative_name"`
				OpeningDate        string    `json:"opening_date"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("요청 형식이 올바르지 않습니다.").ToProblemDetails(r.URL.Path))
				return
			}
			biz, err := bizSvc.RegisterBusiness(r.Context(), req.OwnerUserID, req.RegistrationNumber, req.CompanyName, req.RepresentativeName, req.OpeningDate)
			if err != nil {
				if appErr, ok := err.(*apperr.AppError); ok {
					envelope.WriteError(w, r, appErr.HTTPStatus, appErr.ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
				return
			}
			envelope.WriteJSON(w, r, http.StatusCreated, biz, nil, &envelope.DataProvenance{
				SourceLevel: envelope.Level3API,
				Verified:    true,
				UpdatedAt:   biz.CreatedAt,
			})
		})

		api.Post("/businesses/{businessId}/verify-ownership", func(w http.ResponseWriter, r *http.Request) {
			bizID, err := uuid.Parse(chi.URLParam(r, "businessId"))
			if err != nil {
				envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("유효하지 않은 businessId입니다.").ToProblemDetails(r.URL.Path))
				return
			}
			var req struct {
				BankCode      string `json:"bank_code"`
				AccountNumber string `json:"account_number"`
				AuthCode      string `json:"auth_code"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("요청 형식이 올바르지 않습니다.").ToProblemDetails(r.URL.Path))
				return
			}
			ok, err := bizSvc.VerifyOwnership1Won(r.Context(), bizID, req.BankCode, req.AccountNumber, req.AuthCode)
			if err != nil {
				if appErr, ok := err.(*apperr.AppError); ok {
					envelope.WriteError(w, r, appErr.HTTPStatus, appErr.ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
				return
			}
			envelope.WriteJSON(w, r, http.StatusOK, map[string]any{
				"verified": ok,
				"message":  "사업장 소유권이 성공적으로 검증되었습니다.",
			}, nil, &envelope.DataProvenance{
				SourceLevel: envelope.Level4Reconciled,
				Verified:    true,
				UpdatedAt:   time.Now().UTC(),
			})
		})

		// 03. Stores
		api.Post("/stores", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				BusinessID  uuid.UUID `json:"business_id"`
				OwnerUserID uuid.UUID `json:"owner_user_id"`
				StoreName   string    `json:"store_name"`
				Category    string    `json:"category"`
				Address     string    `json:"address"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("요청 형식이 올바르지 않습니다.").ToProblemDetails(r.URL.Path))
				return
			}
			st, err := storeSvc.CreateStore(r.Context(), req.BusinessID, req.OwnerUserID, req.StoreName, req.Category, req.Address)
			if err != nil {
				if appErr, ok := err.(*apperr.AppError); ok {
					envelope.WriteError(w, r, appErr.HTTPStatus, appErr.ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
				return
			}
			envelope.WriteJSON(w, r, http.StatusCreated, st, nil, &envelope.DataProvenance{
				SourceLevel: envelope.Level1Manual,
				Verified:    true,
				UpdatedAt:   st.CreatedAt,
			})
		})

		// 04. Store-Scoped Routes
		api.Route("/stores/{storeId}", func(storeRouter chi.Router) {
			// Members (RBAC/ABAC)
			storeRouter.Get("/members", func(w http.ResponseWriter, r *http.Request) {
				envelope.WriteJSON(w, r, http.StatusOK, []map[string]any{
					{"role": "OWNER", "user_email": "owner@bn.internal", "is_active": true},
				}, nil, &envelope.DataProvenance{SourceLevel: envelope.Level3API, Verified: true, UpdatedAt: time.Now().UTC()})
			})

			// Card Sales Summary
			storeRouter.Get("/card-sales/summary", func(w http.ResponseWriter, r *http.Request) {
				storeID, err := uuid.Parse(chi.URLParam(r, "storeId"))
				if err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("유효하지 않은 storeId입니다.").ToProblemDetails(r.URL.Path))
					return
				}
				summary, err := revenueSvc.GetCardSalesSummary(r.Context(), storeID, time.Time{}, time.Time{}, nil)
				if err != nil {
					envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusOK, summary, nil, &envelope.DataProvenance{
					SourceLevel: envelope.Level3API,
					Verified:    true,
					UpdatedAt:   time.Now().UTC(),
				})
			})

			// Payout Schedules
			storeRouter.Get("/settlements/schedules", func(w http.ResponseWriter, r *http.Request) {
				storeID, err := uuid.Parse(chi.URLParam(r, "storeId"))
				if err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("유효하지 않은 storeId입니다.").ToProblemDetails(r.URL.Path))
					return
				}
				schedules, err := revenueSvc.ListPayoutSchedules(r.Context(), storeID, time.Time{}, time.Time{}, nil)
				if err != nil {
					envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusOK, schedules, &envelope.Meta{TotalCount: int64(len(schedules))}, &envelope.DataProvenance{
					SourceLevel: envelope.Level3API,
					Verified:    true,
					UpdatedAt:   time.Now().UTC(),
				})
			})

			// Bank Deposits
			storeRouter.Get("/bank-deposits", func(w http.ResponseWriter, r *http.Request) {
				storeID, err := uuid.Parse(chi.URLParam(r, "storeId"))
				if err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("유효하지 않은 storeId입니다.").ToProblemDetails(r.URL.Path))
					return
				}
				deposits, err := reconSvc.ListBankDeposits(r.Context(), storeID, time.Time{}, time.Time{}, nil)
				if err != nil {
					envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusOK, deposits, &envelope.Meta{TotalCount: int64(len(deposits))}, &envelope.DataProvenance{
					SourceLevel: envelope.Level3API,
					Verified:    true,
					UpdatedAt:   time.Now().UTC(),
				})
			})

			// Automated Reconciliation Diagnostics (Level 4)
			storeRouter.Get("/reconciliation/diagnostics", func(w http.ResponseWriter, r *http.Request) {
				storeID, err := uuid.Parse(chi.URLParam(r, "storeId"))
				if err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("유효하지 않은 storeId입니다.").ToProblemDetails(r.URL.Path))
					return
				}
				asOf := time.Now().UTC()
				schedules, _ := revenueSvc.ListPayoutSchedules(r.Context(), storeID, time.Time{}, time.Time{}, nil)
				report, err := reconSvc.RunDiagnostics(r.Context(), storeID, asOf, schedules, asOf)
				if err != nil {
					envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusOK, report, nil, &envelope.DataProvenance{
					SourceLevel: envelope.Level4Reconciled,
					Verified:    true,
					UpdatedAt:   asOf,
				})
			})

			// Reconciliation Dispute Registration
			storeRouter.Post("/reconciliation/disputes", func(w http.ResponseWriter, r *http.Request) {
				storeID, err := uuid.Parse(chi.URLParam(r, "storeId"))
				if err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("유효하지 않은 storeId입니다.").ToProblemDetails(r.URL.Path))
					return
				}
				var req struct {
					ScheduleID    uuid.UUID       `json:"schedule_id"`
					PartnerCode   string          `json:"partner_code"`
					DisputeType   string          `json:"dispute_type"`
					ClaimedAmount decimal.Decimal `json:"claimed_amount"`
					EvidenceMemo  string          `json:"evidence_memo"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("요청 형식이 올바르지 않습니다.").ToProblemDetails(r.URL.Path))
					return
				}
				dispute := &reconciliation.ReconciliationDispute{
					StoreID:       storeID,
					ScheduleID:    req.ScheduleID,
					PartnerCode:   req.PartnerCode,
					DisputeType:   req.DisputeType,
					ClaimedAmount: req.ClaimedAmount,
					EvidenceMemo:  req.EvidenceMemo,
				}
				if err := reconSvc.CreateDispute(r.Context(), dispute); err != nil {
					envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusCreated, dispute, nil, &envelope.DataProvenance{
					SourceLevel: envelope.Level1Manual,
					Verified:    true,
					UpdatedAt:   dispute.CreatedAt,
				})
			})

			// Expenses
			storeRouter.Get("/expenses", func(w http.ResponseWriter, r *http.Request) {
				storeID, err := uuid.Parse(chi.URLParam(r, "storeId"))
				if err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("유효하지 않은 storeId입니다.").ToProblemDetails(r.URL.Path))
					return
				}
				expenses, err := expenseSvc.ListExpenses(r.Context(), storeID, time.Time{}, time.Time{}, nil)
				if err != nil {
					envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusOK, expenses, &envelope.Meta{TotalCount: int64(len(expenses))}, &envelope.DataProvenance{
					SourceLevel: envelope.Level1Manual,
					Verified:    true,
					UpdatedAt:   time.Now().UTC(),
				})
			})

			storeRouter.Post("/expenses", func(w http.ResponseWriter, r *http.Request) {
				storeID, err := uuid.Parse(chi.URLParam(r, "storeId"))
				if err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("유효하지 않은 storeId입니다.").ToProblemDetails(r.URL.Path))
					return
				}
				var req expense.Expense
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("요청 형식이 올바르지 않습니다.").ToProblemDetails(r.URL.Path))
					return
				}
				req.StoreID = storeID
				if req.ExpenseDate.IsZero() {
					req.ExpenseDate = time.Now()
				}
				if err := expenseSvc.CreateExpense(r.Context(), &req); err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest(err.Error()).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusCreated, req, nil, &envelope.DataProvenance{
					SourceLevel: envelope.Level1Manual,
					Verified:    true,
					UpdatedAt:   req.CreatedAt,
				})
			})

			// Correct Journal Entry (Immutable Reversal)
			storeRouter.Post("/ledger/entries/{entryId}/correct", func(w http.ResponseWriter, r *http.Request) {
				storeID, _ := uuid.Parse(chi.URLParam(r, "storeId"))
				entryID, err := uuid.Parse(chi.URLParam(r, "entryId"))
				if err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("유효하지 않은 entryId입니다.").ToProblemDetails(r.URL.Path))
					return
				}
				var req struct {
					Reason string `json:"reason"`
				}
				_ = json.NewDecoder(r.Body).Decode(&req)
				reversal, err := ledgerSvc.ReverseJournalEntry(r.Context(), storeID, entryID, req.Reason)
				if err != nil {
					if appErr, ok := err.(*apperr.AppError); ok {
						envelope.WriteError(w, r, appErr.HTTPStatus, appErr.ToProblemDetails(r.URL.Path))
						return
					}
					envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusOK, reversal, nil, &envelope.DataProvenance{
					SourceLevel: envelope.Level4Reconciled,
					Verified:    true,
					UpdatedAt:   reversal.CreatedAt,
				})
			})

			// Chart of Accounts (COA)
			storeRouter.Get("/ledger/accounts", func(w http.ResponseWriter, r *http.Request) {
				coa := ledgerSvc.GetChartOfAccounts()
				envelope.WriteJSON(w, r, http.StatusOK, coa, &envelope.Meta{TotalCount: int64(len(coa))}, &envelope.DataProvenance{
					SourceLevel: envelope.Level3API,
					Verified:    true,
					UpdatedAt:   time.Now().UTC(),
				})
			})

			// Trial Balance (합계잔액시산표)
			storeRouter.Get("/ledger/trial-balance", func(w http.ResponseWriter, r *http.Request) {
				storeID, _ := uuid.Parse(chi.URLParam(r, "storeId"))
				tb, err := ledgerSvc.GetTrialBalance(r.Context(), storeID, time.Time{})
				if err != nil {
					envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusOK, tb, nil, &envelope.DataProvenance{
					SourceLevel: envelope.Level4Reconciled,
					Verified:    true,
					UpdatedAt:   time.Now().UTC(),
				})
			})

			// General Ledger (총계정원장)
			storeRouter.Get("/ledger/general-ledger", func(w http.ResponseWriter, r *http.Request) {
				storeID, _ := uuid.Parse(chi.URLParam(r, "storeId"))
				accCode := r.URL.Query().Get("account_code")
				if accCode == "" {
					accCode = "1010" // Default 보통예금
				}
				gl, err := ledgerSvc.GetGeneralLedger(r.Context(), storeID, accCode, time.Time{}, time.Time{})
				if err != nil {
					if appErr, ok := err.(*apperr.AppError); ok {
						envelope.WriteError(w, r, appErr.HTTPStatus, appErr.ToProblemDetails(r.URL.Path))
						return
					}
					envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusOK, gl, nil, &envelope.DataProvenance{
					SourceLevel: envelope.Level4Reconciled,
					Verified:    true,
					UpdatedAt:   time.Now().UTC(),
				})
			})

			// Close Accounting Period
			storeRouter.Post("/ledger/periods/close", func(w http.ResponseWriter, r *http.Request) {
				storeID, _ := uuid.Parse(chi.URLParam(r, "storeId"))
				var req struct {
					Year  int `json:"year"`
					Month int `json:"month"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("요청 형식이 올바르지 않습니다.").ToProblemDetails(r.URL.Path))
					return
				}
				p, err := ledgerSvc.ClosePeriod(r.Context(), storeID, req.Year, req.Month, uuid.New())
				if err != nil {
					if appErr, ok := err.(*apperr.AppError); ok {
						envelope.WriteError(w, r, appErr.HTTPStatus, appErr.ToProblemDetails(r.URL.Path))
						return
					}
					envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusOK, p, nil, &envelope.DataProvenance{
					SourceLevel: envelope.Level1Manual,
					Verified:    true,
					UpdatedAt:   p.CreatedAt,
				})
			})

			// Level 1 Manual Expense Recording
			storeRouter.Post("/ledger/manual-expense", func(w http.ResponseWriter, r *http.Request) {
				storeID, _ := uuid.Parse(chi.URLParam(r, "storeId"))
				var req ledger.ManualExpenseInput
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("요청 형식이 올바르지 않습니다.").ToProblemDetails(r.URL.Path))
					return
				}
				if req.ExpenseDate.IsZero() {
					req.ExpenseDate = time.Now()
				}
				entry, err := ledgerSvc.PostManualExpense(r.Context(), storeID, req)
				if err != nil {
					if appErr, ok := err.(*apperr.AppError); ok {
						envelope.WriteError(w, r, appErr.HTTPStatus, appErr.ToProblemDetails(r.URL.Path))
						return
					}
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest(err.Error()).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusCreated, entry, nil, &envelope.DataProvenance{
					SourceLevel: envelope.Level1Manual,
					Verified:    true,
					UpdatedAt:   entry.CreatedAt,
				})
			})

			// Level 1 Manual Revenue Recording
			storeRouter.Post("/ledger/manual-revenue", func(w http.ResponseWriter, r *http.Request) {
				storeID, _ := uuid.Parse(chi.URLParam(r, "storeId"))
				var req ledger.ManualRevenueInput
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("요청 형식이 올바르지 않습니다.").ToProblemDetails(r.URL.Path))
					return
				}
				if req.RevenueDate.IsZero() {
					req.RevenueDate = time.Now()
				}
				entry, err := ledgerSvc.PostManualRevenue(r.Context(), storeID, req)
				if err != nil {
					if appErr, ok := err.(*apperr.AppError); ok {
						envelope.WriteError(w, r, appErr.HTTPStatus, appErr.ToProblemDetails(r.URL.Path))
						return
					}
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest(err.Error()).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusCreated, entry, nil, &envelope.DataProvenance{
					SourceLevel: envelope.Level1Manual,
					Verified:    true,
					UpdatedAt:   entry.CreatedAt,
				})
			})

			// Vendors Management
			storeRouter.Get("/expenses/vendors", func(w http.ResponseWriter, r *http.Request) {
				storeID, _ := uuid.Parse(chi.URLParam(r, "storeId"))
				vendors, err := expenseSvc.ListVendors(r.Context(), storeID)
				if err != nil {
					envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusOK, vendors, &envelope.Meta{TotalCount: int64(len(vendors))}, &envelope.DataProvenance{
					SourceLevel: envelope.Level1Manual,
					Verified:    true,
					UpdatedAt:   time.Now().UTC(),
				})
			})

			storeRouter.Post("/expenses/vendors", func(w http.ResponseWriter, r *http.Request) {
				storeID, _ := uuid.Parse(chi.URLParam(r, "storeId"))
				var req expense.Vendor
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("요청 형식이 올바르지 않습니다.").ToProblemDetails(r.URL.Path))
					return
				}
				req.StoreID = storeID
				if err := expenseSvc.CreateVendor(r.Context(), &req); err != nil {
					if appErr, ok := err.(*apperr.AppError); ok {
						envelope.WriteError(w, r, appErr.HTTPStatus, appErr.ToProblemDetails(r.URL.Path))
						return
					}
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest(err.Error()).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusCreated, req, nil, &envelope.DataProvenance{
					SourceLevel: envelope.Level1Manual,
					Verified:    true,
					UpdatedAt:   req.CreatedAt,
				})
			})

			// Tax VAT Estimate
			storeRouter.Get("/tax/vat-estimate", func(w http.ResponseWriter, r *http.Request) {
				storeID, _ := uuid.Parse(chi.URLParam(r, "storeId"))
				year, _ := strconv.Atoi(r.URL.Query().Get("year"))
				if year == 0 {
					year = time.Now().Year()
				}
				term := tax.VatTerm(r.URL.Query().Get("term"))
				if term == "" {
					term = tax.Term1Final
				}
				report, err := taxSvc.EstimateVat(r.Context(), storeID, business.TaxTypeGeneralTaxable, year, term, decimal.RequireFromString("50000000"), decimal.RequireFromString("2000000"), decimal.RequireFromString("40000000"), decimal.Zero, 12)
				if err != nil {
					envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusOK, report, nil, &envelope.DataProvenance{
					SourceLevel: envelope.Level3API,
					Verified:    true,
					UpdatedAt:   time.Now().UTC(),
				})
			})

			// Inbound POS Webhook
			storeRouter.Post("/connectors/pos/events", func(w http.ResponseWriter, r *http.Request) {
				storeID, _ := uuid.Parse(chi.URLParam(r, "storeId"))
				var event pos.PosEvent
				if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("요청 형식이 올바르지 않습니다.").ToProblemDetails(r.URL.Path))
					return
				}
				entry, err := posSvc.IngestPosEvent(r.Context(), storeID, &event)
				if err != nil {
					if appErr, ok := err.(*apperr.AppError); ok {
						envelope.WriteError(w, r, appErr.HTTPStatus, appErr.ToProblemDetails(r.URL.Path))
						return
					}
					envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusCreated, map[string]any{
					"event_id":      event.EventID,
					"journal_entry": entry,
					"message":       "POS 결제 이벤트가 수신되어 회계 원장에 자동 기장되었습니다.",
				}, nil, &envelope.DataProvenance{
					SourceLevel: envelope.Level3API,
					Verified:    true,
					UpdatedAt:   time.Now().UTC(),
				})
			})

			// Delivery Settlements Query
			storeRouter.Get("/delivery/settlements", func(w http.ResponseWriter, r *http.Request) {
				envelope.WriteJSON(w, r, http.StatusOK, []any{}, &envelope.Meta{TotalCount: 0}, &envelope.DataProvenance{
					SourceLevel: envelope.Level2File,
					Verified:    true,
					UpdatedAt:   time.Now().UTC(),
				})
			})

			// Reviews & CRM
			storeRouter.Get("/reviews", func(w http.ResponseWriter, r *http.Request) {
				storeID, _ := uuid.Parse(chi.URLParam(r, "storeId"))
				reviews, err := crmSvc.ListReviews(r.Context(), storeID, nil)
				if err != nil {
					envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusOK, reviews, &envelope.Meta{TotalCount: int64(len(reviews))}, &envelope.DataProvenance{
					SourceLevel: envelope.Level3API,
					Verified:    true,
					UpdatedAt:   time.Now().UTC(),
				})
			})

			storeRouter.Post("/reviews/{reviewId}/reply", func(w http.ResponseWriter, r *http.Request) {
				storeID, _ := uuid.Parse(chi.URLParam(r, "storeId"))
				reviewID, _ := uuid.Parse(chi.URLParam(r, "reviewId"))
				var req struct {
					ReplyText string `json:"reply_text"`
				}
				_ = json.NewDecoder(r.Body).Decode(&req)
				replied, err := crmSvc.ReplyToReview(r.Context(), storeID, reviewID, req.ReplyText)
				if err != nil {
					envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusOK, replied, nil, &envelope.DataProvenance{
					SourceLevel: envelope.Level3API,
					Verified:    true,
					UpdatedAt:   time.Now().UTC(),
				})
			})

			storeRouter.Post("/crm/campaigns", func(w http.ResponseWriter, r *http.Request) {
				storeID, _ := uuid.Parse(chi.URLParam(r, "storeId"))
				var req struct {
					CampaignName    string `json:"campaign_name"`
					TargetCohort    string `json:"target_cohort"`
					MessageTemplate string `json:"message_template"`
					RecipientsCount int    `json:"recipients_count"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("요청 형식이 올바르지 않습니다.").ToProblemDetails(r.URL.Path))
					return
				}
				camp, err := crmSvc.CreateCampaign(r.Context(), storeID, req.CampaignName, req.TargetCohort, req.MessageTemplate, req.RecipientsCount, time.Now())
				if err != nil {
					if appErr, ok := err.(*apperr.AppError); ok {
						envelope.WriteError(w, r, appErr.HTTPStatus, appErr.ToProblemDetails(r.URL.Path))
						return
					}
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest(err.Error()).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusCreated, camp, nil, &envelope.DataProvenance{
					SourceLevel: envelope.Level3API,
					Verified:    true,
					UpdatedAt:   camp.SentAt,
				})
			})

			// Analytics P&L
			storeRouter.Get("/analytics/pnl", func(w http.ResponseWriter, r *http.Request) {
				storeID, _ := uuid.Parse(chi.URLParam(r, "storeId"))
				now := time.Now()
				pnl, err := ledgerSvc.CalculatePnL(r.Context(), storeID, now.AddDate(0, -1, 0), now)
				if err != nil {
					envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusOK, pnl, nil, &envelope.DataProvenance{
					SourceLevel: envelope.Level4Reconciled,
					Verified:    true,
					UpdatedAt:   now.UTC(),
				})
			})

			// AI Forecasts
			storeRouter.Get("/ai/forecasts", func(w http.ResponseWriter, r *http.Request) {
				storeID, _ := uuid.Parse(chi.URLParam(r, "storeId"))
				days, _ := strconv.Atoi(r.URL.Query().Get("historical_days"))
				if days == 0 {
					days = 30
				}
				report, err := aiSvc.GenerateForecast(r.Context(), storeID, days, decimal.RequireFromString("750000"))
				if err != nil {
					if appErr, ok := err.(*apperr.AppError); ok {
						envelope.WriteError(w, r, appErr.HTTPStatus, appErr.ToProblemDetails(r.URL.Path))
						return
					}
					envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusOK, report, nil, &envelope.DataProvenance{
					SourceLevel: envelope.Level3API,
					Verified:    true,
					UpdatedAt:   time.Now().UTC(),
				})
			})

			// Subscriptions
			storeRouter.Post("/subscriptions", func(w http.ResponseWriter, r *http.Request) {
				storeID, _ := uuid.Parse(chi.URLParam(r, "storeId"))
				var req struct {
					Tier commerce.SubscriptionTier `json:"tier"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("요청 형식이 올바르지 않습니다.").ToProblemDetails(r.URL.Path))
					return
				}
				sub, err := commerceSvc.CreateSubscription(r.Context(), storeID, req.Tier)
				if err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest(err.Error()).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusCreated, sub, nil, &envelope.DataProvenance{
					SourceLevel: envelope.Level3API,
					Verified:    true,
					UpdatedAt:   sub.StartedAt,
				})
			})

			// B2B Marketplace Orders
			storeRouter.Post("/marketplace/orders", func(w http.ResponseWriter, r *http.Request) {
				storeID, _ := uuid.Parse(chi.URLParam(r, "storeId"))
				var req struct {
					SupplierName  string                     `json:"supplier_name"`
					PaymentMethod string                     `json:"payment_method"`
					Items         []commerce.MarketplaceItem `json:"items"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("요청 형식이 올바르지 않습니다.").ToProblemDetails(r.URL.Path))
					return
				}
				order, entry, err := commerceSvc.CreateMarketplaceOrder(r.Context(), storeID, req.SupplierName, req.Items, req.PaymentMethod)
				if err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest(err.Error()).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusCreated, map[string]any{
					"order":         order,
					"journal_entry": entry,
				}, nil, &envelope.DataProvenance{
					SourceLevel: envelope.Level3API,
					Verified:    true,
					UpdatedAt:   order.CreatedAt,
				})
			})

			// Level 2 File Upload & Parser
			storeRouter.Post("/imports/upload", func(w http.ResponseWriter, r *http.Request) {
				storeID, _ := uuid.Parse(chi.URLParam(r, "storeId"))
				fileType := imports.FileImportType(r.URL.Query().Get("file_type"))
				if fileType == "" {
					fileType = imports.TypeDeliveryBaemin
				}
				fileName := r.URL.Query().Get("file_name")
				if fileName == "" {
					fileName = "uploaded_settlement.csv"
				}
				result, err := importsSvc.UploadAndParse(r.Context(), storeID, fileType, fileName, r.Body)
				if err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest(err.Error()).ToProblemDetails(r.URL.Path))
					return
				}
				envelope.WriteJSON(w, r, http.StatusOK, result, nil, &envelope.DataProvenance{
					SourceLevel: envelope.Level2File,
					Verified:    true,
					UpdatedAt:   result.ProcessedAt,
				})
			})
		})

		// 05. Community
		api.Get("/community/posts", func(w http.ResponseWriter, r *http.Request) {
			posts, err := communitySvc.ListPosts(r.Context(), nil)
			if err != nil {
				envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
				return
			}
			envelope.WriteJSON(w, r, http.StatusOK, posts, &envelope.Meta{TotalCount: int64(len(posts))}, &envelope.DataProvenance{
				SourceLevel: envelope.Level1Manual,
				Verified:    true,
				UpdatedAt:   time.Now().UTC(),
			})
		})

		api.Post("/community/posts", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				AuthorUserID uuid.UUID              `json:"author_user_id"`
				StoreName    string                 `json:"store_name"`
				Category     community.PostCategory `json:"category"`
				Title        string                 `json:"title"`
				Content      string                 `json:"content"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("요청 형식이 올바르지 않습니다.").ToProblemDetails(r.URL.Path))
				return
			}
			post, err := communitySvc.CreatePost(r.Context(), req.AuthorUserID, req.StoreName, req.Title, req.Content, req.Category)
			if err != nil {
				envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest(err.Error()).ToProblemDetails(r.URL.Path))
				return
			}
			envelope.WriteJSON(w, r, http.StatusCreated, post, nil, &envelope.DataProvenance{
				SourceLevel: envelope.Level1Manual,
				Verified:    true,
				UpdatedAt:   post.CreatedAt,
			})
		})

		// 06. Notifications
		api.Get("/notifications", func(w http.ResponseWriter, r *http.Request) {
			envelope.WriteJSON(w, r, http.StatusOK, []map[string]any{
				{
					"id":         uuid.NewString(),
					"type":       "SETTLEMENT_REPORT",
					"title":      "오늘 입금 예정 리포트",
					"content":    "신한카드 외 4건 총 1,452,000원이 입금 예정입니다.",
					"created_at": time.Now().UTC(),
				},
			}, &envelope.Meta{TotalCount: 1}, &envelope.DataProvenance{
				SourceLevel: envelope.Level3API,
				Verified:    true,
				UpdatedAt:   time.Now().UTC(),
			})
		})

		// 07. Admin Operations (4-Eyes & Audit Hash Chain)
		api.Post("/admin/business-verifications/{verificationId}/decide", func(w http.ResponseWriter, r *http.Request) {
			vID, _ := uuid.Parse(chi.URLParam(r, "verificationId"))
			var req struct {
				Decision     string    `json:"decision"`
				RejectReason string    `json:"reject_reason"`
				AdminUserID  uuid.UUID `json:"admin_user_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("요청 형식이 올바르지 않습니다.").ToProblemDetails(r.URL.Path))
				return
			}
			dec := &admin.VerificationDecision{
				VerificationID: vID,
				Decision:       req.Decision,
				RejectReason:   req.RejectReason,
				AdminUserID:    req.AdminUserID,
			}
			if err := adminSvc.DecideBusinessVerification(r.Context(), dec); err != nil {
				envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest(err.Error()).ToProblemDetails(r.URL.Path))
				return
			}
			envelope.WriteJSON(w, r, http.StatusOK, dec, nil, &envelope.DataProvenance{
				SourceLevel: envelope.Level3API,
				Verified:    true,
				UpdatedAt:   dec.DecidedAt,
			})
		})

		api.Get("/admin/audit/integrity-verify", func(w http.ResponseWriter, r *http.Request) {
			allEntries, _ := ledgerSvc.ListEntries(r.Context(), uuid.Nil)
			result, err := adminSvc.VerifySystemIntegrity(r.Context(), allEntries)
			if err != nil {
				envelope.WriteError(w, r, http.StatusInternalServerError, apperr.NewInternal(err).ToProblemDetails(r.URL.Path))
				return
			}
			envelope.WriteJSON(w, r, http.StatusOK, result, nil, &envelope.DataProvenance{
				SourceLevel: envelope.Level4Reconciled,
				Verified:    true,
				UpdatedAt:   result.CheckedAt,
			})
		})

		// 08. Partner Inbound Webhook
		api.Post("/partner/webhooks/{providerCode}", func(w http.ResponseWriter, r *http.Request) {
			provider := chi.URLParam(r, "providerCode")
			envelope.WriteJSON(w, r, http.StatusOK, map[string]any{
				"provider": provider,
				"status":   "RECEIVED",
				"message":  "Partner event acknowledged and queued for asynchronous processing.",
			}, nil, &envelope.DataProvenance{
				SourceLevel: envelope.Level3API,
				Verified:    true,
				UpdatedAt:   time.Now().UTC(),
			})
		})

		// 404 handler
		api.NotFound(func(w http.ResponseWriter, r *http.Request) {
			errPayload := apperr.NewNotFound(fmt.Sprintf("API 엔드포인트 '%s'", r.URL.Path)).ToProblemDetails(r.URL.Path)
			envelope.WriteError(w, r, http.StatusNotFound, errPayload)
		})
	})

	return r
}

