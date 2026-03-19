package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ReactionKind string

const (
	Upvote   ReactionKind = "upvote"
	Downvote ReactionKind = "downvote"
)

type Post struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	AuthorID   uuid.UUID `json:"authorId" gorm:"type:uuid;index;not null"`
	Pseudonym  string    `json:"pseudonym" gorm:"not null"`
	Content    string    `json:"content" gorm:"type:text;not null"`
	ReplyCount int       `json:"replyCount" gorm:"not null;default:0"`
	Score      int       `json:"score" gorm:"not null;default:0"`
	CreatedAt  time.Time `json:"createdAt"`
}

type Reply struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	PostID    uuid.UUID `json:"postId" gorm:"type:uuid;index;not null"`
	AuthorID  uuid.UUID `json:"authorId" gorm:"type:uuid;index;not null"`
	Pseudonym string    `json:"pseudonym" gorm:"not null"`
	Content   string    `json:"content" gorm:"type:text;not null"`
	CreatedAt time.Time `json:"createdAt"`
}

type Reaction struct {
	UserID    uuid.UUID    `json:"userId" gorm:"type:uuid;primaryKey"`
	PostID    uuid.UUID    `json:"postId" gorm:"type:uuid;primaryKey"`
	Kind      ReactionKind `json:"kind" gorm:"type:text;not null"`
	CreatedAt time.Time    `json:"createdAt"`
}

type FeedCursor struct {
	CreatedAt time.Time
}

type PostRepository interface {
	Create(ctx context.Context, post *Post) error
	DeleteByAuthor(ctx context.Context, postID, authorID uuid.UUID) error
	GetByID(ctx context.Context, postID uuid.UUID) (*Post, error)
	CreateReply(ctx context.Context, reply *Reply) error
	ListReplies(ctx context.Context, postID uuid.UUID, limit int) ([]Reply, error)
	UpsertReaction(ctx context.Context, postID, userID uuid.UUID, kind ReactionKind) error
}

type PostService interface {
	Create(ctx context.Context, authorID uuid.UUID, pseudonym, content string) (*Post, error)
	Delete(ctx context.Context, postID, authorID uuid.UUID) error
	GetByID(ctx context.Context, postID uuid.UUID) (*Post, error)
	React(ctx context.Context, postID, userID uuid.UUID, kind ReactionKind) error
	CreateReply(ctx context.Context, postID, authorID uuid.UUID, pseudonym, content string) (*Reply, error)
	ListReplies(ctx context.Context, postID uuid.UUID, limit int) ([]Reply, error)
}

type FeedRepository interface {
	Latest(ctx context.Context, limit int, cursor *FeedCursor) ([]Post, *FeedCursor, error)
	Trending(ctx context.Context, limit int) ([]Post, error)
}

type FeedService interface {
	Latest(ctx context.Context, limit int, cursor *FeedCursor) ([]Post, *FeedCursor, error)
	Trending(ctx context.Context, limit int) ([]Post, error)
}
