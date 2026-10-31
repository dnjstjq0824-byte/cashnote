package community

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestCommunityService(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	authorID := uuid.New()

	t.Run("Create and list community posts", func(t *testing.T) {
		post, err := svc.CreatePost(ctx, authorID, "마포구 베이커리 사장님", "소상공인 전기요금 특별지원 신청 팁", "소진공 사이트에서 간편인증 후 바로 신청 가능합니다.", CategoryFunds)
		if err != nil {
			t.Fatalf("failed to create post: %v", err)
		}

		if post.Title != "소상공인 전기요금 특별지원 신청 팁" {
			t.Errorf("title mismatch: %s", post.Title)
		}

		posts, err := svc.ListPosts(ctx, nil)
		if err != nil || len(posts) != 1 {
			t.Fatalf("expected 1 post, got %d", len(posts))
		}
	})
}

