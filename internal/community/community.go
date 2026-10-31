package community

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	apperr "merchant-os/internal/platform/errors"
)

// PostCategory represents community topic
type PostCategory string

const (
	CategoryKnowhow PostCategory = "BUSINESS_KNOWHOW"
	CategoryFunds   PostCategory = "POLICY_FUNDS"
	CategoryQna     PostCategory = "QNA"
)

// CommunityPost represents a post in verified merchant community
type CommunityPost struct {
	ID              uuid.UUID    `json:"id"`
	AuthorUserID    uuid.UUID    `json:"author_user_id"`
	AuthorStoreName string       `json:"author_store_name"` // Anonymous display name e.g. "강남구 카페 사장님"
	Category        PostCategory `json:"category"`
	Title           string       `json:"title"`
	Content         string       `json:"content"`
	LikeCount       int          `json:"like_count"`
	CommentCount    int          `json:"comment_count"`
	CreatedAt       time.Time    `json:"created_at"`
}

// Service manages merchant community
type Service interface {
	CreatePost(ctx context.Context, authorID uuid.UUID, storeName, title, content string, category PostCategory) (*CommunityPost, error)
	ListPosts(ctx context.Context, category *PostCategory) ([]*CommunityPost, error)
}

type serviceImpl struct {
	mu    sync.RWMutex
	posts []*CommunityPost
}

func NewService() Service {
	return &serviceImpl{}
}

func (s *serviceImpl) CreatePost(ctx context.Context, authorID uuid.UUID, storeName, title, content string, category PostCategory) (*CommunityPost, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if title == "" || content == "" {
		return nil, apperr.NewBadRequest("게시글 제목과 내용을 모두 입력해 주세요.")
	}

	post := &CommunityPost{
		ID:              uuid.New(),
		AuthorUserID:    authorID,
		AuthorStoreName: storeName,
		Category:        category,
		Title:           title,
		Content:         content,
		LikeCount:       0,
		CommentCount:    0,
		CreatedAt:       time.Now().UTC(),
	}

	s.posts = append(s.posts, post)
	return post, nil
}

func (s *serviceImpl) ListPosts(ctx context.Context, category *PostCategory) ([]*CommunityPost, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*CommunityPost
	for _, p := range s.posts {
		if category != nil && p.Category != *category {
			continue
		}
		result = append(result, p)
	}
	return result, nil
}

