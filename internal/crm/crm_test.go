package crm

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCrmService(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	storeID := uuid.New()

	t.Run("Review sentiment and Google API reply vs Baemin Draft for Copy", func(t *testing.T) {
		revGoogle := &Review{
			StoreID:    storeID,
			Platform:   PlatformGoogle,
			AuthorName: "Alice",
			Rating:     5,
			Content:    "음식이 너무 맛있고 친절해요!",
		}
		_ = svc.AddReview(ctx, revGoogle)

		revBaemin := &Review{
			StoreID:    storeID,
			Platform:   PlatformBaemin,
			AuthorName: "Bob",
			Rating:     2,
			Content:    "배달이 30분 늦게 왔습니다.",
		}
		_ = svc.AddReview(ctx, revBaemin)

		if revGoogle.Sentiment != SentimentPositive {
			t.Errorf("expected 5-star to be POSITIVE, got %s", revGoogle.Sentiment)
		}
		if revBaemin.Sentiment != SentimentNegative {
			t.Errorf("expected 2-star to be NEGATIVE, got %s", revBaemin.Sentiment)
		}

		// Google Reply -> direct API
		repliedGoogle, err := svc.ReplyToReview(ctx, storeID, revGoogle.ID, "감사합니다!")
		if err != nil || repliedGoogle.IsDraftCopy {
			t.Errorf("Google review reply should be direct API (IsDraftCopy=false)")
		}

		// Baemin Reply -> Draft for Copy
		repliedBaemin, err := svc.ReplyToReview(ctx, storeID, revBaemin.ID, "")
		if err != nil || !repliedBaemin.IsDraftCopy {
			t.Errorf("Baemin review reply must be draft for clipboard copy (IsDraftCopy=true)")
		}
		if repliedBaemin.ReplyText == "" {
			t.Errorf("expected AI auto-generated apology draft")
		}
	})

	t.Run("Night-time marketing campaign ban (20:50 ~ 08:00)", func(t *testing.T) {
		loc := time.FixedZone("KST", 9*60*60)
		// 21:00 KST (Night time)
		nightTime := time.Date(2026, 10, 31, 21, 0, 0, 0, loc)
		_, err := svc.CreateCampaign(ctx, storeID, "야간 할인 쿠폰", "ALL", "지금 주문하면 20% 할인!", 100, nightTime)
		if err == nil {
			t.Fatalf("night-time marketing campaign must be rejected")
		}

		// 14:00 KST (Day time)
		dayTime := time.Date(2026, 10, 31, 14, 0, 0, 0, loc)
		campaign, err := svc.CreateCampaign(ctx, storeID, "오후 할인 쿠폰", "ALL", "오후 티타임 10% 할인!", 100, dayTime)
		if err != nil {
			t.Fatalf("day-time campaign should succeed: %v", err)
		}
		if campaign.Status != "SENT" {
			t.Errorf("expected campaign status SENT, got %s", campaign.Status)
		}
	})
}
