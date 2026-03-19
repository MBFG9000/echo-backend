package service

import (
	"context"
	"strings"

	"github.com/echo-app/echo/internal/domain"
)

type Post struct {
	posts domain.PostRepository
}

func NewPost(posts domain.PostRepository) *Post {
	return &Post{posts: posts}
}

func (p *Post) Create(ctx context.Context, userID uint, content string) (*domain.Post, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || len(trimmed) > 280 {
		return nil, domain.ErrInvalidInput
	}

	post := &domain.Post{
		UserID:  userID,
		Content: trimmed,
	}

	if err := p.posts.Create(ctx, post); err != nil {
		return nil, err
	}

	return post, nil
}

func (p *Post) ListLatest(ctx context.Context, limit int) ([]domain.Post, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return p.posts.ListLatest(ctx, limit)
}
