package admin

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"merchant-os/internal/ledger"
)

func TestAdminService(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	t.Run("4-Eyes business verification decision", func(t *testing.T) {
		vID := uuid.New()
		adminID := uuid.New()
		err := svc.DecideBusinessVerification(ctx, &VerificationDecision{
			VerificationID: vID,
			Decision:       "APPROVED",
			AdminUserID:    adminID,
		})
		if err != nil {
			t.Fatalf("failed to decide verification: %v", err)
		}
	})

	t.Run("System integrity check verifies balanced ledger and produces hash fingerprint", func(t *testing.T) {
		entries := []*ledger.JournalEntry{
			{
				ID: uuid.New(),
				Lines: []ledger.JournalLine{
					{AccountCode: "1020", Debit: decimal.RequireFromString("100000"), Credit: decimal.Zero},
					{AccountCode: "4010", Debit: decimal.Zero, Credit: decimal.RequireFromString("100000")},
				},
				CreatedAt: time.Now(),
			},
		}

		result, err := svc.VerifySystemIntegrity(ctx, entries)
		if err != nil {
			t.Fatalf("failed to verify integrity: %v", err)
		}

		if !result.DoubleEntryLedgerBalanced {
			t.Errorf("expected balanced ledger")
		}
		if result.UnbalancedEntriesCount != 0 {
			t.Errorf("expected 0 unbalanced entries")
		}
		if result.Verdict != "VERIFIED_PASS" {
			t.Errorf("expected VERIFIED_PASS, got %s", result.Verdict)
		}
		if result.IntegrityFingerprint == "" {
			t.Errorf("expected non-empty cryptographic hash fingerprint")
		}
	})
}
