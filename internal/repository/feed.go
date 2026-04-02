package repository

import (
	"context"
	"time"

	"github.com/echo-app/echo/internal/domain"
	"gorm.io/gorm"
)

type Feed struct {
	db *gorm.DB
}

func NewFeed(db *gorm.DB) *Feed {
	return &Feed{db: db}
}

func (f *Feed) Latest(ctx context.Context, limit int, cursor *domain.FeedCursor) ([]domain.Post, *domain.FeedCursor, error) {
	query := f.db.WithContext(ctx).
		Where("is_hidden = false").
		Order("created_at DESC").
		Order("id DESC").
		Limit(limit)

	if cursor != nil && !cursor.CreatedAt.IsZero() {
		query = query.Where("created_at < ?", cursor.CreatedAt)
	}

	posts := make([]domain.Post, 0, limit)
	if err := query.Find(&posts).Error; err != nil {
		return nil, nil, err
	}

	if len(posts) == 0 {
		return posts, nil, nil
	}

	next := &domain.FeedCursor{CreatedAt: posts[len(posts)-1].CreatedAt.UTC().Truncate(time.Microsecond)}
	return posts, next, nil
}

func (f *Feed) Trending(ctx context.Context, limit int) ([]domain.Post, error) {
	posts := make([]domain.Post, 0, limit)
	err := f.db.WithContext(ctx).
		Where("is_hidden = false").
		Order("score DESC").
		Order("created_at DESC").
		Limit(limit).
		Find(&posts).Error
	if err != nil {
		return nil, err
	}

	return posts, nil
}
