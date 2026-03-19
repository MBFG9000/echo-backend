package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/echo-app/echo/internal/domain"
	"github.com/redis/go-redis/v9"
)

type Feed struct {
	feeds domain.FeedRepository
	redis *redis.Client
}

func NewFeed(feeds domain.FeedRepository, redisClient *redis.Client) *Feed {
	return &Feed{feeds: feeds, redis: redisClient}
}

func (f *Feed) Latest(ctx context.Context, limit int, cursor *domain.FeedCursor) ([]domain.Post, *domain.FeedCursor, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return f.feeds.Latest(ctx, limit, cursor)
}

func (f *Feed) Trending(ctx context.Context, limit int) ([]domain.Post, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	cacheKey := "feed:trending:" + strconv.Itoa(limit)
	cached, err := f.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		posts := make([]domain.Post, 0, limit)
		if json.Unmarshal([]byte(cached), &posts) == nil {
			return posts, nil
		}
	}
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("get trending cache: %w", err)
	}

	posts, err := f.feeds.Trending(ctx, limit)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(posts)
	if err == nil {
		_ = f.redis.Set(ctx, cacheKey, string(payload), 60*time.Second).Err()
	}

	return posts, nil
}

func ParseCursor(raw string) (*domain.FeedCursor, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	createdAt, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	return &domain.FeedCursor{CreatedAt: createdAt}, nil
}
