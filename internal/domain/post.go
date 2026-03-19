package domain

import (
	"context"
	"time"
)

type Post struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"userId" gorm:"index;not null"`
	Content   string    `json:"content" gorm:"type:text;not null"`
	CreatedAt time.Time `json:"createdAt"`
	User      User      `json:"user" gorm:"foreignKey:UserID"`
}

type PostRepository interface {
	Create(ctx context.Context, post *Post) error
	ListLatest(ctx context.Context, limit int) ([]Post, error)
}

type PostService interface {
	Create(ctx context.Context, userID uint, content string) (*Post, error)
	ListLatest(ctx context.Context, limit int) ([]Post, error)
}
