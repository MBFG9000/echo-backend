package repository

import (
	"context"

	"github.com/echo-app/echo/internal/domain"
	"gorm.io/gorm"
)

type Post struct {
	db *gorm.DB
}

func NewPost(db *gorm.DB) *Post {
	return &Post{db: db}
}

func (p *Post) Create(ctx context.Context, post *domain.Post) error {
	return p.db.WithContext(ctx).Create(post).Error
}

func (p *Post) ListLatest(ctx context.Context, limit int) ([]domain.Post, error) {
	posts := make([]domain.Post, 0, limit)
	err := p.db.WithContext(ctx).
		Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Find(&posts).Error
	if err != nil {
		return nil, err
	}

	return posts, nil
}
