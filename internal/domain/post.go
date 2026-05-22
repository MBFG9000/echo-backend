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
	ID         uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey"`
	AuthorID   uuid.UUID       `json:"authorId" gorm:"type:uuid;index;not null"`
	Pseudonym  string          `json:"pseudonym" gorm:"not null"`
	Content    string          `json:"content" gorm:"type:text;not null"`
	Attachment *PostAttachment `json:"attachment,omitempty" gorm:"foreignKey:PostID"`
	IsHidden   bool            `json:"-" gorm:"not null;default:false"`
	ReplyCount int             `json:"replyCount" gorm:"not null;default:0"`
	Score      int             `json:"score" gorm:"not null;default:0"`
	CreatedAt  time.Time       `json:"createdAt"`
}

type PostAttachment struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	PostID      uuid.UUID `json:"postId" gorm:"type:uuid;uniqueIndex;not null"`
	FileName    string    `json:"fileName" gorm:"not null"`
	ContentType string    `json:"contentType" gorm:"not null"`
	Size        int64     `json:"size" gorm:"not null"`
	Data        []byte    `json:"-" gorm:"type:bytea;not null"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Reply struct {
	ID            uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	PostID        uuid.UUID  `json:"postId" gorm:"type:uuid;index;not null"`
	ParentReplyID *uuid.UUID `json:"parentReplyId,omitempty" gorm:"type:uuid;index"`
	AuthorID      uuid.UUID  `json:"authorId" gorm:"type:uuid;index;not null"`
	Pseudonym     string     `json:"pseudonym" gorm:"not null"`
	Content       string     `json:"content" gorm:"type:text;not null"`
	Score         int        `json:"score" gorm:"not null;default:0"`
	CreatedAt     time.Time  `json:"createdAt"`
	Children      []Reply    `json:"children,omitempty" gorm:"-"`
}

type ReplyReaction struct {
	UserID    uuid.UUID    `json:"userId" gorm:"type:uuid;primaryKey"`
	ReplyID   uuid.UUID    `json:"replyId" gorm:"type:uuid;primaryKey"`
	Kind      ReactionKind `json:"kind" gorm:"type:text;not null"`
	CreatedAt time.Time    `json:"createdAt"`
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
	Search(ctx context.Context, query string, limit int) ([]Post, error)
	GetAttachment(ctx context.Context, attachmentID uuid.UUID) (*PostAttachment, error)
	SetHidden(ctx context.Context, postID uuid.UUID, hidden bool) error
	CreateReply(ctx context.Context, reply *Reply) error
	ListReplies(ctx context.Context, postID uuid.UUID, limit int) ([]Reply, error)
	UpdateReplyByAuthor(ctx context.Context, replyID, authorID uuid.UUID, content string) (*Reply, error)
	DeleteReplyByAuthor(ctx context.Context, replyID, authorID uuid.UUID) error
	UpsertReplyReaction(ctx context.Context, replyID, userID uuid.UUID, kind ReactionKind) error
	UpsertReaction(ctx context.Context, postID, userID uuid.UUID, kind ReactionKind) error
}

type PostService interface {
	Create(ctx context.Context, authorID uuid.UUID, pseudonym, content string, attachment *PostAttachment) (*Post, error)
	Delete(ctx context.Context, postID, authorID uuid.UUID) error
	GetByID(ctx context.Context, postID uuid.UUID) (*Post, error)
	Search(ctx context.Context, query string, limit int) ([]Post, error)
	GetAttachment(ctx context.Context, attachmentID uuid.UUID) (*PostAttachment, error)
	React(ctx context.Context, postID, userID uuid.UUID, kind ReactionKind) error
	CreateReply(ctx context.Context, postID uuid.UUID, parentReplyID *uuid.UUID, authorID uuid.UUID, pseudonym, content string) (*Reply, error)
	ListReplies(ctx context.Context, postID uuid.UUID, limit int) ([]Reply, error)
	UpdateReply(ctx context.Context, replyID, authorID uuid.UUID, content string) (*Reply, error)
	DeleteReply(ctx context.Context, replyID, authorID uuid.UUID) error
	ReactReply(ctx context.Context, replyID, userID uuid.UUID, kind ReactionKind) error
}

type FeedRepository interface {
	Latest(ctx context.Context, limit int, cursor *FeedCursor) ([]Post, *FeedCursor, error)
	Trending(ctx context.Context, limit int) ([]Post, error)
}

type FeedService interface {
	Latest(ctx context.Context, limit int, cursor *FeedCursor) ([]Post, *FeedCursor, error)
	Trending(ctx context.Context, limit int) ([]Post, error)
}

type ReportService interface {
	Create(ctx context.Context, postID, reporterID uuid.UUID, reason string) (bool, error)
	ListOpen(ctx context.Context, limit, offset int) ([]Report, error)
	Act(ctx context.Context, adminID, reportID uuid.UUID, action ModerationAction, note string) error
}
