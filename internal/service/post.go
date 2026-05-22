package service

import (
	"context"
	"strings"

	"github.com/echo-app/echo/internal/domain"
	"github.com/echo-app/echo/internal/realtime"
	"github.com/google/uuid"
)

type Post struct {
	posts     domain.PostRepository
	publisher *realtime.Publisher
}

func NewPost(posts domain.PostRepository, publisher *realtime.Publisher) *Post {
	return &Post{posts: posts, publisher: publisher}
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

	if err := p.posts.DeleteByAuthor(ctx, postID, authorID); err != nil {
		return err
	}

	_ = p.publisher.Publish(ctx, realtime.EventPostDeleted, realtime.PostIDPayload{PostID: postID.String()})
	return nil
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

	if err := p.posts.UpsertReaction(ctx, postID, userID, kind); err != nil {
		return err
	}

	p.publishPostUpdated(ctx, postID)
	return nil
}

func (p *Post) Unreact(ctx context.Context, postID, userID uuid.UUID) error {
	if _, err := p.posts.GetByID(ctx, postID); err != nil {
		return err
	}

	if err := p.posts.DeleteReaction(ctx, postID, userID); err != nil {
		return err
	}

	p.publishPostUpdated(ctx, postID)
	return nil
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

	_ = p.publisher.Publish(ctx, realtime.EventReplyCreated, reply)
	p.publishPostUpdated(ctx, postID)
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

	reply, err := p.posts.UpdateReplyByAuthor(ctx, replyID, authorID, trimmed)
	if err != nil {
		return nil, err
	}

	_ = p.publisher.Publish(ctx, realtime.EventReplyUpdated, reply)
	return reply, nil
}

func (p *Post) DeleteReply(ctx context.Context, replyID, authorID uuid.UUID) error {
	reply, err := p.posts.GetReplyByID(ctx, replyID)
	if err != nil {
		return err
	}

	if err := p.posts.DeleteReplyByAuthor(ctx, replyID, authorID); err != nil {
		return err
	}

	_ = p.publisher.Publish(ctx, realtime.EventReplyDeleted, realtime.ReplyDeletedPayload{
		PostID:  reply.PostID.String(),
		ReplyID: replyID.String(),
	})
	p.publishPostUpdated(ctx, reply.PostID)
	return nil
}

func (p *Post) ReactReply(ctx context.Context, replyID, userID uuid.UUID, kind domain.ReactionKind) error {
	if kind != domain.Upvote && kind != domain.Downvote {
		return domain.ErrInvalidInput
	}

	if err := p.posts.UpsertReplyReaction(ctx, replyID, userID, kind); err != nil {
		return err
	}

	p.publishReplyScore(ctx, replyID)
	return nil
}

func (p *Post) UnreactReply(ctx context.Context, replyID, userID uuid.UUID) error {
	if err := p.posts.DeleteReplyReaction(ctx, replyID, userID); err != nil {
		return err
	}

	p.publishReplyScore(ctx, replyID)
	return nil
}

func (p *Post) publishPostUpdated(ctx context.Context, postID uuid.UUID) {
	post, err := p.posts.GetByID(ctx, postID)
	if err != nil {
		return
	}

	_ = p.publisher.Publish(ctx, realtime.EventPostUpdated, realtime.PostUpdatedPayload{
		PostID:     post.ID.String(),
		Score:      post.Score,
		ReplyCount: post.ReplyCount,
	})
}

func (p *Post) publishReplyScore(ctx context.Context, replyID uuid.UUID) {
	reply, err := p.posts.GetReplyByID(ctx, replyID)
	if err != nil {
		return
	}

	_ = p.publisher.Publish(ctx, realtime.EventReplyUpdated, reply)
}

func (p *Post) MarkViewerReactionsOnPosts(ctx context.Context, userID uuid.UUID, posts []domain.Post) {
	if len(posts) == 0 {
		return
	}

	ids := make([]uuid.UUID, len(posts))
	for i := range posts {
		ids[i] = posts[i].ID
	}

	liked, err := p.posts.LikedPostIDsAmong(ctx, userID, ids)
	if err != nil {
		return
	}

	for i := range posts {
		posts[i].LikedByMe = liked[posts[i].ID]
	}
}

func (p *Post) MarkViewerReactionsOnReplies(ctx context.Context, userID uuid.UUID, replies []domain.Reply) {
	ids := collectReplyIDs(replies)
	if len(ids) == 0 {
		return
	}

	liked, err := p.posts.LikedReplyIDsAmong(ctx, userID, ids)
	if err != nil {
		return
	}

	markRepliesLiked(replies, liked)
}

func collectReplyIDs(replies []domain.Reply) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(replies))
	var walk func([]domain.Reply)
	walk = func(items []domain.Reply) {
		for _, reply := range items {
			ids = append(ids, reply.ID)
			if len(reply.Children) > 0 {
				walk(reply.Children)
			}
		}
	}
	walk(replies)

	return ids
}

func markRepliesLiked(replies []domain.Reply, liked map[uuid.UUID]bool) {
	for i := range replies {
		replies[i].LikedByMe = liked[replies[i].ID]
		if len(replies[i].Children) > 0 {
			markRepliesLiked(replies[i].Children, liked)
		}
	}
}
