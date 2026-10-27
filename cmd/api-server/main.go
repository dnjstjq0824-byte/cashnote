package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
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

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := chi.NewRouter()

	// Middlewares
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

		// 01. Authentication & Identity
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
				SourceLevel: envelope.Level3API, // NTS Verified
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
			// Financial P&L
			storeRouter.Get("/analytics/pnl", func(w http.ResponseWriter, r *http.Request) {
				storeID, err := uuid.Parse(chi.URLParam(r, "storeId"))
				if err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("유효하지 않은 storeId입니다.").ToProblemDetails(r.URL.Path))
					return
				}
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

			// Level 2 File Upload Parser (Delivery Settlements)
			storeRouter.Post("/delivery/settlements/parse-csv", func(w http.ResponseWriter, r *http.Request) {
				storeID, err := uuid.Parse(chi.URLParam(r, "storeId"))
				if err != nil {
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest("유효하지 않은 storeId입니다.").ToProblemDetails(r.URL.Path))
					return
				}
				platformStr := r.URL.Query().Get("platform")
				platform := parsers.PlatformBaemin
				if platformStr == "COUPANG_EATS" {
					platform = parsers.PlatformCoupangEats
				} else if platformStr == "YOGIYO" {
					platform = parsers.PlatformYogiyo
				}

				records, err := parsers.ParseDeliverySettlementCSV(storeID, platform, r.Body)
				if err != nil {
					if appErr, ok := err.(*apperr.AppError); ok {
						envelope.WriteError(w, r, appErr.HTTPStatus, appErr.ToProblemDetails(r.URL.Path))
						return
					}
					envelope.WriteError(w, r, http.StatusBadRequest, apperr.NewBadRequest(err.Error()).ToProblemDetails(r.URL.Path))
					return
				}

				envelope.WriteJSON(w, r, http.StatusOK, records, &envelope.Meta{
					TotalCount: int64(len(records)),
				}, &envelope.DataProvenance{
					SourceLevel: envelope.Level2File,
					Verified:    true,
					UpdatedAt:   time.Now().UTC(),
				})
			})
		})

		// 404 handler
		api.NotFound(func(w http.ResponseWriter, r *http.Request) {
			errPayload := apperr.NewNotFound(fmt.Sprintf("API 엔드포인트 '%s'", r.URL.Path)).ToProblemDetails(r.URL.Path)
			envelope.WriteError(w, r, http.StatusNotFound, errPayload)
		})
	})

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("BN Merchant OS Core API Server running on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server Listen error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down API server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced shutdown: %v", err)
	}
	log.Println("API server stopped.")
}
