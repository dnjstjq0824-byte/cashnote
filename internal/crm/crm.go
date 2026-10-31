package crm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	apperr "merchant-os/internal/platform/errors"
)

// ReviewPlatform represents review source platform
type ReviewPlatform string

const (
	PlatformGoogle ReviewPlatform = "GOOGLE_BUSINESS"
	PlatformNaver  ReviewPlatform = "NAVER_PLACE"
	PlatformBaemin ReviewPlatform = "BAEMIN"
	PlatformCoupang ReviewPlatform = "COUPANG_EATS"
)

// Sentiment represents sentiment classification
type Sentiment string

const (
	SentimentPositive Sentiment = "POSITIVE"
	SentimentNeutral  Sentiment = "NEUTRAL"
	SentimentNegative Sentiment = "NEGATIVE"
)

// Review represents a customer review across multiple platforms
type Review struct {
	ID           uuid.UUID      `json:"id"`
	StoreID      uuid.UUID      `json:"store_id"`
	Platform     ReviewPlatform `json:"platform"`
	AuthorName   string         `json:"author_name"`
	Rating       int            `json:"rating"` // 1 ~ 5
	Content      string         `json:"content"`
	Sentiment    Sentiment      `json:"sentiment"`
	ReplyText    string         `json:"reply_text,omitempty"`
	IsReplied    bool           `json:"is_replied"`
	IsDraftCopy  bool           `json:"is_draft_for_copy"` // true if third party API is blocked
	CreatedAt    time.Time      `json:"created_at"`
}

// CrmCampaign represents customer marketing campaign
type CrmCampaign struct {
	ID              uuid.UUID `json:"id"`
	StoreID         uuid.UUID `json:"store_id"`
	CampaignName    string    `json:"campaign_name"`
	TargetCohort    string    `json:"target_cohort"` // ALL, VIP_REVISIT, DORMANT
	MessageTemplate string    `json:"message_template"`
	TotalRecipients int       `json:"total_recipients"`
	SentAt          time.Time `json:"sent_at"`
	Status          string    `json:"status"` // SENT, SCHEDULED, BLOCKED_NIGHT_TIME
}

// Service manages reviews and CRM campaigns
type Service interface {
	AddReview(ctx context.Context, rev *Review) error
	ListReviews(ctx context.Context, storeID uuid.UUID, platform *ReviewPlatform) ([]*Review, error)
	ReplyToReview(ctx context.Context, storeID, reviewID uuid.UUID, replyText string) (*Review, error)
	CreateCampaign(ctx context.Context, storeID uuid.UUID, name, cohort, template string, recipientCount int, sendAt time.Time) (*CrmCampaign, error)
}

type serviceImpl struct {
	mu        sync.RWMutex
	reviews   map[uuid.UUID][]*Review
	campaigns map[uuid.UUID][]*CrmCampaign
}

func NewService() Service {
	return &serviceImpl{
		reviews:   make(map[uuid.UUID][]*Review),
		campaigns: make(map[uuid.UUID][]*CrmCampaign),
	}
}

func (s *serviceImpl) AddReview(ctx context.Context, rev *Review) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if rev.ID == uuid.Nil {
		rev.ID = uuid.New()
	}

	// Automatic sentiment detection
	if rev.Rating >= 4 {
		rev.Sentiment = SentimentPositive
	} else if rev.Rating == 3 {
		rev.Sentiment = SentimentNeutral
	} else {
		rev.Sentiment = SentimentNegative
	}

	s.reviews[rev.StoreID] = append(s.reviews[rev.StoreID], rev)
	return nil
}

func (s *serviceImpl) ListReviews(ctx context.Context, storeID uuid.UUID, platform *ReviewPlatform) ([]*Review, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := s.reviews[storeID]
	var result []*Review
	for _, r := range list {
		if platform != nil && r.Platform != *platform {
			continue
		}
		result = append(result, r)
	}
	return result, nil
}

// ReplyToReview handles API dispatch for Google and AI Draft for Blocked platforms (Scenario H)
func (s *serviceImpl) ReplyToReview(ctx context.Context, storeID, reviewID uuid.UUID, replyText string) (*Review, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var target *Review
	for _, r := range s.reviews[storeID] {
		if r.ID == reviewID {
			target = r
			break
		}
	}

	if target == nil {
		return nil, apperr.NewNotFound("리뷰")
	}

	if strings.TrimSpace(replyText) == "" {
		// AI Auto-Draft Generation
		if target.Sentiment == SentimentPositive {
			replyText = fmt.Sprintf("안녕하세요 사장님 매장입니다. 소중한 별점 %d점 리뷰 감사드립니다! 항상 정성을 다하겠습니다.", target.Rating)
		} else {
			replyText = "안녕하세요 사장님 매장입니다. 이용 중 불편을 드려 진심으로 사과드립니다. 고객님의 피드백을 반영하여 더 나은 서비스를 제공하겠습니다."
		}
	}

	target.ReplyText = replyText
	target.IsReplied = true

	// Strict regulatory & API access boundary check:
	// Google Business Profile is PUBLIC_API -> direct live reply
	// Naver / Baemin / Coupang are BLOCKED -> return draft for 1-click clipboard copy
	if target.Platform == PlatformGoogle {
		target.IsDraftCopy = false
	} else {
		target.IsDraftCopy = true
	}

	return target, nil
}

// CreateCampaign enforces night-time marketing ban (20:50 ~ 08:00) per Information and Communications Network Act
func (s *serviceImpl) CreateCampaign(ctx context.Context, storeID uuid.UUID, name, cohort, template string, recipientCount int, sendAt time.Time) (*CrmCampaign, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sendAt.IsZero() {
		sendAt = time.Now()
	}

	// KST time check
	loc, _ := time.LoadLocation("Asia/Seoul")
	if loc == nil {
		loc = time.FixedZone("KST", 9*60*60)
	}
	kstTime := sendAt.In(loc)
	hour := kstTime.Hour()
	min := kstTime.Minute()

	// 20:50 ~ 08:00 prohibition
	isNightTime := false
	if hour >= 21 || hour < 8 {
		isNightTime = true
	} else if hour == 20 && min >= 50 {
		isNightTime = true
	}

	if isNightTime {
		return nil, apperr.NewBadRequest("정보통신망법 제50조에 따라 야간 시간대(20:50 ~ 08:00)에는 영리목적의 마케팅 알림톡/문자 발송이 법적으로 금지됩니다.")
	}

	campaign := &CrmCampaign{
		ID:              uuid.New(),
		StoreID:         storeID,
		CampaignName:    name,
		TargetCohort:    cohort,
		MessageTemplate: template,
		TotalRecipients: recipientCount,
		SentAt:          sendAt,
		Status:          "SENT",
	}

	s.campaigns[storeID] = append(s.campaigns[storeID], campaign)
	return campaign, nil
}

