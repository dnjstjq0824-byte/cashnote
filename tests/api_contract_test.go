package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"merchant-os/internal/platform/envelope"
	"merchant-os/internal/platform/server"
)

func TestMasterAPIContractSuite(t *testing.T) {
	handler := server.NewRouter()

	// Global variables carried forward across user journey
	var userID uuid.UUID
	var businessID uuid.UUID
	var storeID uuid.UUID

	t.Run("01. Health & Capabilities", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/health/live", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/v1/system/capabilities", nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/v1/system/monorepo", nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("02. User Authentication & Profile", func(t *testing.T) {
		// Register
		regJSON := `{"email":"ceo@cafe-bn.internal","password":"SecurePassword123!","full_name":"김대표"}`
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader([]byte(regJSON)))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("register failed: %d, body: %s", rec.Code, rec.Body.String())
		}

		var regResp envelope.Response
		_ = json.Unmarshal(rec.Body.Bytes(), &regResp)
		dataMap := regResp.Data.(map[string]any)
		userID = uuid.MustParse(dataMap["id"].(string))

		// Login
		loginJSON := `{"email":"ceo@cafe-bn.internal","password":"SecurePassword123!"}`
		req = httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader([]byte(loginJSON)))
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("login failed: %d", rec.Code)
		}

		// User Profile Me
		req = httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("get me failed: %d", rec.Code)
		}

		// Consent Revoke
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/users/me/consents/%s/revoke", uuid.NewString()), nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("consent revoke failed: %d", rec.Code)
		}
	})

	t.Run("03. Business Registration & 1-Won Verification", func(t *testing.T) {
		bizJSON := fmt.Sprintf(`{"owner_user_id":"%s","registration_number":"123-45-67890","company_name":"BN카페 강남본점","representative_name":"김대표","opening_date":"20240101"}`, userID)
		req := httptest.NewRequest(http.MethodPost, "/v1/businesses", bytes.NewReader([]byte(bizJSON)))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("business reg failed: %d, body: %s", rec.Code, rec.Body.String())
		}

		var bizResp envelope.Response
		_ = json.Unmarshal(rec.Body.Bytes(), &bizResp)
		bizMap := bizResp.Data.(map[string]any)
		businessID = uuid.MustParse(bizMap["id"].(string))

		// 1-won verification
		verifyJSON := `{"bank_code":"004","account_number":"110-123","auth_code":"7788"}`
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/businesses/%s/verify-ownership", businessID), bytes.NewReader([]byte(verifyJSON)))
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("verify ownership failed: %d", rec.Code)
		}
	})

	t.Run("04. Store Creation & Store Members", func(t *testing.T) {
		storeJSON := fmt.Sprintf(`{"business_id":"%s","owner_user_id":"%s","store_name":"BN카페 역삼역점","category":"카페","address":"서울 강남구 테헤란로 152"}`, businessID, userID)
		req := httptest.NewRequest(http.MethodPost, "/v1/stores", bytes.NewReader([]byte(storeJSON)))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("store create failed: %d, body: %s", rec.Code, rec.Body.String())
		}

		var storeResp envelope.Response
		_ = json.Unmarshal(rec.Body.Bytes(), &storeResp)
		stMap := storeResp.Data.(map[string]any)
		storeID = uuid.MustParse(stMap["id"].(string))

		// List members
		req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/stores/%s/members", storeID), nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("list store members failed: %d", rec.Code)
		}
	})

	t.Run("05. Card Sales Summary & Settlements Payout Schedules", func(t *testing.T) {
		// Summary
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/stores/%s/card-sales/summary", storeID), nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("card sales summary failed: %d", rec.Code)
		}

		// Schedules
		req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/stores/%s/settlements/schedules", storeID), nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("payout schedules failed: %d", rec.Code)
		}
	})

	t.Run("06. Bank Deposits, Automated Reconciliation & Disputes", func(t *testing.T) {
		// Bank deposits query
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/stores/%s/bank-deposits", storeID), nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("bank deposits query failed: %d", rec.Code)
		}

		// Reconciliation diagnostics (Level 4)
		req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/stores/%s/reconciliation/diagnostics", storeID), nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("reconciliation diagnostics failed: %d", rec.Code)
		}

		// Dispute creation
		disputeJSON := fmt.Sprintf(`{"schedule_id":"%s","partner_code":"CARD_SHINHAN","dispute_type":"UNPAID_DEFAULT","claimed_amount":"150000","evidence_memo":"미입금 소명 요청"}`, uuid.NewString())
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/stores/%s/reconciliation/disputes", storeID), bytes.NewReader([]byte(disputeJSON)))
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("reconciliation dispute failed: %d", rec.Code)
		}
	})

	t.Run("07. Expenses Management", func(t *testing.T) {
		// Create expense
		expJSON := `{"category_code":"FIXED_RENT","vendor_name":"역삼빌딩","supply_amount":"3000000","vat_amount":"300000","payment_method":"BANK_TRANSFER","proof_type":"TAX_INVOICE","memo":"10월 임차료"}`
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/stores/%s/expenses", storeID), bytes.NewReader([]byte(expJSON)))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create expense failed: %d, body: %s", rec.Code, rec.Body.String())
		}

		// List expenses
		req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/stores/%s/expenses", storeID), nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("list expenses failed: %d", rec.Code)
		}
	})

	t.Run("08. POS Event Ingestion & Automatic Journal Posting", func(t *testing.T) {
		posJSON := `{"event_id":"pos-evt-101","pos_vendor":"PAYHERE","event_type":"ORDER_PAID","pos_order_number":"ORD-101","order_time":"2026-10-31T12:00:00Z","payments":[{"method":"CARD","amount":"55000"}],"total_amount":"55000"}`
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/stores/%s/connectors/pos/events", storeID), bytes.NewReader([]byte(posJSON)))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("pos event ingestion failed: %d, body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("09. Level 2 Delivery File Upload & Parsing", func(t *testing.T) {
		csvData := "정산일자,주문금액,취소금액,중개이용료,배달대행료,결제수수료,할인쿠폰,광고비,수수료부가세\n2026-10-31,250000,0,17000,22500,7500,2000,10000,5900\n"
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/stores/%s/imports/upload?file_type=DELIVERY_BAEMIN", storeID), bytes.NewReader([]byte(csvData)))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("imports upload failed: %d, body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("10. Analytics P&L & Tax VAT Estimation", func(t *testing.T) {
		// P&L
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/stores/%s/analytics/pnl", storeID), nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("analytics pnl failed: %d", rec.Code)
		}

		// VAT estimate
		req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/stores/%s/tax/vat-estimate?year=2026&term=2_FINAL", storeID), nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("vat estimate failed: %d", rec.Code)
		}
	})

	t.Run("11. Reviews & CRM Campaigns", func(t *testing.T) {
		// List reviews
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/stores/%s/reviews", storeID), nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("list reviews failed: %d", rec.Code)
		}

		// CRM Daytime campaign
		crmJSON := `{"campaign_name":"주말 브런치 쿠폰","target_cohort":"ALL","message_template":"주말 브런치 10% 할인 쿠폰","recipients_count":50}`
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/stores/%s/crm/campaigns", storeID), bytes.NewReader([]byte(crmJSON)))
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		// Check either success or night time rejection depending on current KST
		loc := time.FixedZone("KST", 9*60*60)
		nowKST := time.Now().In(loc)
		if nowKST.Hour() >= 21 || nowKST.Hour() < 8 {
			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400 rejection during night-time hours, got %d", rec.Code)
			}
		} else {
			if rec.Code != http.StatusCreated {
				t.Errorf("expected 201 during daytime, got %d body: %s", rec.Code, rec.Body.String())
			}
		}
	})

	t.Run("12. AI Forecasts with Data Sufficiency", func(t *testing.T) {
		// Sufficient data
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/stores/%s/ai/forecasts?historical_days=30", storeID), nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("ai forecasts failed: %d, body: %s", rec.Code, rec.Body.String())
		}

		// Insufficient data (<14 days)
		req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/stores/%s/ai/forecasts?historical_days=7", storeID), nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422 for insufficient data, got %d", rec.Code)
		}
	})

	t.Run("13. Subscriptions & B2B Marketplace Orders", func(t *testing.T) {
		// Plus tier subscription
		subJSON := `{"tier":"PLUS"}`
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/stores/%s/subscriptions", storeID), bytes.NewReader([]byte(subJSON)))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("subscription failed: %d", rec.Code)
		}

		// Marketplace Order
		orderJSON := `{"supplier_name":"CJ프레시웨이","payment_method":"PREPAID","items":[{"item_name":"우유 1L 10팩","quantity":2,"unit_price":"25000"}]}`
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/stores/%s/marketplace/orders", storeID), bytes.NewReader([]byte(orderJSON)))
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("marketplace order failed: %d, body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("14. Community Posts & Notifications", func(t *testing.T) {
		postJSON := fmt.Sprintf(`{"author_user_id":"%s","store_name":"강남 카페 사장님","category":"BUSINESS_KNOWHOW","title":"배달 포장비 절감 노하우","content":"벌크 구매 추천드립니다."}`, userID)
		req := httptest.NewRequest(http.MethodPost, "/v1/community/posts", bytes.NewReader([]byte(postJSON)))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create post failed: %d", rec.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/v1/community/posts", nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("list posts failed: %d", rec.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/v1/notifications", nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("notifications failed: %d", rec.Code)
		}
	})

	t.Run("15. Admin 4-Eyes Decisions & System Cryptographic Integrity", func(t *testing.T) {
		adminJSON := fmt.Sprintf(`{"decision":"APPROVED","admin_user_id":"%s"}`, uuid.NewString())
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/admin/business-verifications/%s/decide", uuid.NewString()), bytes.NewReader([]byte(adminJSON)))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("admin decision failed: %d", rec.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/v1/admin/audit/integrity-verify", nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("audit integrity verify failed: %d", rec.Code)
		}
	})

	t.Run("16. Partner Inbound Webhook", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/partner/webhooks/TOSSPAYMENTS", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("partner webhook failed: %d", rec.Code)
		}
	})

	t.Run("17. Phase 2 Level 1 Manual Accounting, Trial Balance & Period Closing", func(t *testing.T) {
		// 1. Chart of Accounts
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/stores/%s/ledger/accounts", storeID), nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("get accounts failed: %d", rec.Code)
		}

		// 2. Create Vendor
		vendorJSON := `{"trade_name":"(주)에스프레소원두","business_number":"123-45-67890","category":"원두유통","representative":"이원두"}`
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/stores/%s/expenses/vendors", storeID), bytes.NewReader([]byte(vendorJSON)))
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create vendor failed: %d, body: %s", rec.Code, rec.Body.String())
		}

		// 3. Post Manual Expense (Level 1)
		expJSON := `{"expense_date":"2026-02-01T10:00:00Z","account_code":"5010","payment_method":"BANK","supply_amount":"150000","vat_amount":"15000","vendor_name":"(주)에스프레소원두","description":"원두 5kg 매입"}`
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/stores/%s/ledger/manual-expense", storeID), bytes.NewReader([]byte(expJSON)))
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("manual expense failed: %d, body: %s", rec.Code, rec.Body.String())
		}

		// 4. Post Manual Revenue (Level 1)
		revJSON := `{"revenue_date":"2026-02-01T12:00:00Z","channel":"STORE","payment_method":"BANK","supply_amount":"300000","vat_amount":"30000","description":"오후 홀 매출"}`
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/stores/%s/ledger/manual-revenue", storeID), bytes.NewReader([]byte(revJSON)))
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("manual revenue failed: %d, body: %s", rec.Code, rec.Body.String())
		}

		// 5. Get Trial Balance
		req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/stores/%s/ledger/trial-balance", storeID), nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("get trial balance failed: %d, body: %s", rec.Code, rec.Body.String())
		}

		var tbResp envelope.Response
		_ = json.Unmarshal(rec.Body.Bytes(), &tbResp)
		tbMap := tbResp.Data.(map[string]any)
		if tbMap["is_balanced"] != true {
			t.Fatalf("trial balance must be balanced: %v", tbMap)
		}

		// 6. Get General Ledger for 1010 (보통예금)
		req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/stores/%s/ledger/general-ledger?account_code=1010", storeID), nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("get general ledger failed: %d, body: %s", rec.Code, rec.Body.String())
		}

		// 7. Close Accounting Period (2026년 1월 마감)
		closeJSON := `{"year":2026,"month":1}`
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/stores/%s/ledger/periods/close", storeID), bytes.NewReader([]byte(closeJSON)))
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("close period failed: %d, body: %s", rec.Code, rec.Body.String())
		}
	})
}
