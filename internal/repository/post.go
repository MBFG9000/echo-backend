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
	if post.Attachment != nil {
		if post.Attachment.ID == uuid.Nil {
			post.Attachment.ID = uuid.New()
		}
		post.Attachment.PostID = post.ID
		if post.Attachment.CreatedAt.IsZero() {
			post.Attachment.CreatedAt = time.Now()
		}
	}

	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Attachment").Create(post).Error; err != nil {
			return err
		}
		if post.Attachment != nil {
			return tx.Create(post.Attachment).Error
		}

		return nil
	})
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
	err := p.db.WithContext(ctx).
		Preload("Attachment", attachmentMetadataScope).
		Where("id = ? AND is_hidden = false", postID).
		First(&post).Error
	if err == nil {
		return &post, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}

	return nil, err
}

func (p *Post) GetAttachment(ctx context.Context, attachmentID uuid.UUID) (*domain.PostAttachment, error) {
	var attachment domain.PostAttachment
	err := p.db.WithContext(ctx).
		Joins("JOIN posts ON posts.id = post_attachments.post_id").
		Where("post_attachments.id = ? AND posts.is_hidden = false", attachmentID).
		First(&attachment).Error
	if err == nil {
		return &attachment, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}

	return nil, err
}

func (p *Post) Search(ctx context.Context, query string, limit int) ([]domain.Post, error) {
	posts := make([]domain.Post, 0, limit)
	err := p.db.WithContext(ctx).
		Where("is_hidden = false").
		Where("content ILIKE ? OR pseudonym ILIKE ?", "%"+query+"%", "%"+query+"%").
		Order("created_at DESC").
		Limit(limit).
		Find(&posts).Error
	if err != nil {
		return nil, err
	}

	return posts, nil
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
	if reply.ParentReplyID != nil {
		parent, err := p.getReplyByID(ctx, *reply.ParentReplyID)
		if err != nil {
			return err
		}
		if parent.PostID != reply.PostID {
			return domain.ErrInvalidInput
		}
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

func (p *Post) UpdateReplyByAuthor(ctx context.Context, replyID, authorID uuid.UUID, content string) (*domain.Reply, error) {
	reply, err := p.getReplyByID(ctx, replyID)
	if err != nil {
		return nil, err
	}

	if reply.AuthorID != authorID {
		return nil, domain.ErrUnauthorized
	}

	if err := p.db.WithContext(ctx).
		Model(&domain.Reply{}).
		Where("id = ?", replyID).
		Update("content", content).Error; err != nil {
		return nil, err
	}

	reply.Content = content
	return reply, nil
}

func (p *Post) DeleteReplyByAuthor(ctx context.Context, replyID, authorID uuid.UUID) error {
	reply, err := p.getReplyByID(ctx, replyID)
	if err != nil {
		return err
	}

	if reply.AuthorID != authorID {
		return domain.ErrUnauthorized
	}

	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var deletedCount int64
		if err := tx.Raw(`
			WITH RECURSIVE subtree AS (
				SELECT id FROM replies WHERE id = ?
				UNION ALL
				SELECT r.id FROM replies r
				INNER JOIN subtree s ON r.parent_reply_id = s.id
			)
			SELECT COUNT(*) FROM subtree;
		`, replyID).Scan(&deletedCount).Error; err != nil {
			return err
		}
		if deletedCount == 0 {
			return domain.ErrNotFound
		}

		result := tx.Where("id = ?", replyID).Delete(&domain.Reply{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}

		if err := tx.Model(&domain.Post{}).
			Where("id = ?", reply.PostID).
			UpdateColumn("reply_count", gorm.Expr("GREATEST(reply_count - ?, 0)", deletedCount)).Error; err != nil {
			return err
		}

		return nil
	})
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
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&created)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected > 0 {
				scoreDelta = reactionValue(kind)
			}
		} else if existing.Kind == kind {
			return domain.ErrConflict
		} else {
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

func (p *Post) UpsertReplyReaction(ctx context.Context, replyID, userID uuid.UUID, kind domain.ReactionKind) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var reply domain.Reply
		if err := tx.Where("id = ?", replyID).First(&reply).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}

		var existing domain.ReplyReaction
		err := tx.Where("reply_id = ? AND user_id = ?", replyID, userID).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		scoreDelta := 0
		if errors.Is(err, gorm.ErrRecordNotFound) {
			created := domain.ReplyReaction{ReplyID: replyID, UserID: userID, Kind: kind}
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&created)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected > 0 {
				scoreDelta = reactionValue(kind)
			}
		} else if existing.Kind == kind {
			return domain.ErrConflict
		} else {
			if err := tx.Model(&domain.ReplyReaction{}).
				Where("reply_id = ? AND user_id = ?", replyID, userID).
				Update("kind", kind).Error; err != nil {
				return err
			}
			scoreDelta = reactionValue(kind) - reactionValue(existing.Kind)
		}

		if scoreDelta != 0 {
			if err := tx.Model(&domain.Reply{}).
				Where("id = ?", replyID).
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

func (p *Post) DeleteReaction(ctx context.Context, postID, userID uuid.UUID) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing domain.Reaction
		err := tx.Where("post_id = ? AND user_id = ?", postID, userID).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}

		if err := tx.Delete(&existing).Error; err != nil {
			return err
		}

		return tx.Model(&domain.Post{}).
			Where("id = ?", postID).
			UpdateColumn("score", gorm.Expr("score - ?", reactionValue(existing.Kind))).Error
	})
}

func (p *Post) DeleteReplyReaction(ctx context.Context, replyID, userID uuid.UUID) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing domain.ReplyReaction
		err := tx.Where("reply_id = ? AND user_id = ?", replyID, userID).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}

		if err := tx.Delete(&existing).Error; err != nil {
			return err
		}

		return tx.Model(&domain.Reply{}).
			Where("id = ?", replyID).
			UpdateColumn("score", gorm.Expr("score - ?", reactionValue(existing.Kind))).Error
	})
}
func attachmentMetadataScope(db *gorm.DB) *gorm.DB {
	return db.Select("id", "post_id", "file_name", "content_type", "size", "created_at")
}

func (p *Post) getReplyByID(ctx context.Context, replyID uuid.UUID) (*domain.Reply, error) {
	var reply domain.Reply
	err := p.db.WithContext(ctx).Where("id = ?", replyID).First(&reply).Error
	if err == nil {
		return &reply, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}

	return nil, err
}
