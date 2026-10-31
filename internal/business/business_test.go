package business

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestBusinessService(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	ownerID := uuid.New()

	t.Run("Register business with valid BRN and NTS check", func(t *testing.T) {
		biz, err := svc.RegisterBusiness(ctx, ownerID, "123-45-67890", "맛있는식당", "홍길동", "20240101")
		if err != nil {
			t.Fatalf("failed to register business: %v", err)
		}

		if biz.RegistrationNumber != "1234567890" {
			t.Errorf("expected clean BRN 1234567890, got %s", biz.RegistrationNumber)
		}
		if biz.Status != StatusActive {
			t.Errorf("expected ACTIVE status, got %s", biz.Status)
		}
		if biz.VerificationStatus != VerificationPending {
			t.Errorf("expected PENDING verification status, got %s", biz.VerificationStatus)
		}

		// Verify 1-won ownership with correct code
		ok, err := svc.VerifyOwnership1Won(ctx, biz.ID, "004", "110-123-456789", "7788")
		if err != nil || !ok {
			t.Fatalf("1-won verification failed: %v", err)
		}

		updatedBiz, _ := svc.GetBusinessByID(ctx, biz.ID)
		if updatedBiz.VerificationStatus != VerificationVerified {
			t.Errorf("expected VERIFIED status after 1-won check, got %s", updatedBiz.VerificationStatus)
		}
	})

	t.Run("Invalid BRN format is rejected", func(t *testing.T) {
		_, err := svc.RegisterBusiness(ctx, ownerID, "12345", "불량식당", "아무개", "20240101")
		if err == nil {
			t.Fatalf("expected error for short BRN")
		}
	})

	t.Run("Duplicate BRN is rejected with Conflict", func(t *testing.T) {
		_, err := svc.RegisterBusiness(ctx, ownerID, "123-45-67890", "중복식당", "김철수", "20240101")
		if err == nil {
			t.Fatalf("expected conflict on duplicate BRN")
		}
	})
}

