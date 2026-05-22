package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/echo-app/echo/internal/domain"
	"github.com/google/uuid"
)

type PostBroadcaster interface {
	Broadcast(payload []byte)
}

type Post struct {
	posts       domain.PostRepository
	broadcaster PostBroadcaster
}

func NewPost(posts domain.PostRepository, broadcaster PostBroadcaster) *Post {
	return &Post{posts: posts, broadcaster: broadcaster}
}

func (p *Post) Create(ctx context.Context, authorID uuid.UUID, pseudonym, content string, attachment *domain.PostAttachment) (*domain.Post, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || len(trimmed) > 280 {
		return nil, domain.ErrInvalidInput
	}

	if strings.TrimSpace(pseudonym) == "" {
		return nil, domain.ErrInvalidInput
	}

	post := &domain.Post{
		ID:         uuid.New(),
		AuthorID:   authorID,
		Pseudonym:  pseudonym,
		Content:    trimmed,
		Attachment: attachment,
	}

	if err := p.posts.Create(ctx, post); err != nil {
		return nil, err
	}

	if p.broadcaster != nil {
		if payload, err := json.Marshal(post); err == nil {
			p.broadcaster.Broadcast(payload)
		}
	}

	return post, nil
}

func (p *Post) Delete(ctx context.Context, postID, authorID uuid.UUID) error {
	post, err := p.posts.GetByID(ctx, postID)
	if err != nil {
		return err
	}
	if post.AuthorID != authorID {
		return domain.ErrUnauthorized
	}

	return p.posts.DeleteByAuthor(ctx, postID, authorID)
}

func (p *Post) GetByID(ctx context.Context, postID uuid.UUID) (*domain.Post, error) {
	return p.posts.GetByID(ctx, postID)
}

func (p *Post) GetAttachment(ctx context.Context, attachmentID uuid.UUID) (*domain.PostAttachment, error) {
	return p.posts.GetAttachment(ctx, attachmentID)
}

func (p *Post) Search(ctx context.Context, query string, limit int) ([]domain.Post, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return nil, domain.ErrInvalidInput
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return p.posts.Search(ctx, trimmed, limit)
}

func (p *Post) React(ctx context.Context, postID, userID uuid.UUID, kind domain.ReactionKind) error {
	if kind != domain.Upvote && kind != domain.Downvote {
		return domain.ErrInvalidInput
	}

	if _, err := p.posts.GetByID(ctx, postID); err != nil {
		return err
	}

	return p.posts.UpsertReaction(ctx, postID, userID, kind)
}

func (p *Post) CreateReply(ctx context.Context, postID uuid.UUID, parentReplyID *uuid.UUID, authorID uuid.UUID, pseudonym, content string) (*domain.Reply, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || len(trimmed) > 280 {
		return nil, domain.ErrInvalidInput
	}

	if strings.TrimSpace(pseudonym) == "" {
		return nil, domain.ErrInvalidInput
	}

	if _, err := p.posts.GetByID(ctx, postID); err != nil {
		return nil, err
	}

	reply := &domain.Reply{
		ID:            uuid.New(),
		PostID:        postID,
		ParentReplyID: parentReplyID,
		AuthorID:      authorID,
		Pseudonym:     pseudonym,
		Content:       trimmed,
	}

	if err := p.posts.CreateReply(ctx, reply); err != nil {
		return nil, err
	}

	return reply, nil
}

func (p *Post) ListReplies(ctx context.Context, postID uuid.UUID, limit int) ([]domain.Reply, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	if _, err := p.posts.GetByID(ctx, postID); err != nil {
		return nil, err
	}

	return p.posts.ListReplies(ctx, postID, limit)
}

func (p *Post) UpdateReply(ctx context.Context, replyID, authorID uuid.UUID, content string) (*domain.Reply, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || len(trimmed) > 280 {
		return nil, domain.ErrInvalidInput
	}

	return p.posts.UpdateReplyByAuthor(ctx, replyID, authorID, trimmed)
}

func (p *Post) DeleteReply(ctx context.Context, replyID, authorID uuid.UUID) error {
	return p.posts.DeleteReplyByAuthor(ctx, replyID, authorID)
}

func (p *Post) ReactReply(ctx context.Context, replyID, userID uuid.UUID, kind domain.ReactionKind) error {
	if kind != domain.Upvote && kind != domain.Downvote {
		return domain.ErrInvalidInput
	}

	return p.posts.UpsertReplyReaction(ctx, replyID, userID, kind)
}
