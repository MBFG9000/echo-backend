package repository

import (
	"context"
	"errors"
	"time"

	"github.com/echo-app/echo/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Post struct {
	db *gorm.DB
}

func NewPost(db *gorm.DB) *Post {
	return &Post{db: db}
}

func (p *Post) Create(ctx context.Context, post *domain.Post) error {
	if post.ID == uuid.Nil {
		post.ID = uuid.New()
	}
	if post.CreatedAt.IsZero() {
		post.CreatedAt = time.Now()
	}

	return p.db.WithContext(ctx).Create(post).Error
}

func (p *Post) DeleteByAuthor(ctx context.Context, postID, authorID uuid.UUID) error {
	result := p.db.WithContext(ctx).
		Where("id = ? AND author_id = ?", postID, authorID).
		Delete(&domain.Post{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (p *Post) GetByID(ctx context.Context, postID uuid.UUID) (*domain.Post, error) {
	var post domain.Post
	err := p.db.WithContext(ctx).Where("id = ? AND is_hidden = false", postID).First(&post).Error
	if err == nil {
		return &post, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}

	return nil, err
}

func (p *Post) SetHidden(ctx context.Context, postID uuid.UUID, hidden bool) error {
	result := p.db.WithContext(ctx).Model(&domain.Post{}).
		Where("id = ?", postID).
		Update("is_hidden", hidden)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (p *Post) CreateReply(ctx context.Context, reply *domain.Reply) error {
	if reply.ID == uuid.Nil {
		reply.ID = uuid.New()
	}
	if reply.CreatedAt.IsZero() {
		reply.CreatedAt = time.Now()
	}

	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(reply).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.Post{}).
			Where("id = ?", reply.PostID).
			UpdateColumn("reply_count", gorm.Expr("reply_count + 1")).Error; err != nil {
			return err
		}

		return nil
	})
}

func (p *Post) ListReplies(ctx context.Context, postID uuid.UUID, limit int) ([]domain.Reply, error) {
	replies := make([]domain.Reply, 0, limit)
	err := p.db.WithContext(ctx).
		Where("post_id = ?", postID).
		Order("created_at ASC").
		Limit(limit).
		Find(&replies).Error
	if err != nil {
		return nil, err
	}

	return replies, nil
}

func (p *Post) UpsertReaction(ctx context.Context, postID, userID uuid.UUID, kind domain.ReactionKind) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing domain.Reaction
		err := tx.Where("post_id = ? AND user_id = ?", postID, userID).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		scoreDelta := 0
		if errors.Is(err, gorm.ErrRecordNotFound) {
			created := domain.Reaction{PostID: postID, UserID: userID, Kind: kind}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&created).Error; err != nil {
				return err
			}
			if tx.RowsAffected > 0 {
				scoreDelta = reactionValue(kind)
			}
		} else if existing.Kind != kind {
			if err := tx.Model(&domain.Reaction{}).
				Where("post_id = ? AND user_id = ?", postID, userID).
				Update("kind", kind).Error; err != nil {
				return err
			}
			scoreDelta = reactionValue(kind) - reactionValue(existing.Kind)
		}

		if scoreDelta != 0 {
			if err := tx.Model(&domain.Post{}).
				Where("id = ?", postID).
				UpdateColumn("score", gorm.Expr("score + ?", scoreDelta)).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func reactionValue(kind domain.ReactionKind) int {
	if kind == domain.Upvote {
		return 1
	}

	return -1
}
