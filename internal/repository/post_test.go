package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/echo-app/echo/internal/domain"
	"github.com/google/uuid"
)

func TestPostRepository_CreateGetDelete(t *testing.T) {
	db, cleanup := setupPostgres(t)
	defer cleanup()

	repo := NewPost(db, NewOutbox(db))
	post := &domain.Post{ID: uuid.New(), AuthorID: uuid.New(), Pseudonym: "post-author", Content: "foo"}
	if err := repo.Create(context.Background(), post); err != nil {
		t.Fatalf("create post failed: %v", err)
	}

	fetched, err := repo.GetByID(context.Background(), post.ID)
	if err != nil {
		t.Fatalf("get post failed: %v", err)
	}
	if fetched.ID != post.ID {
		t.Fatalf("expected ID %s got %s", post.ID, fetched.ID)
	}

	if err := repo.DeleteByAuthor(context.Background(), post.ID, post.AuthorID); err != nil {
		t.Fatalf("delete post failed: %v", err)
	}

	if _, err := repo.GetByID(context.Background(), post.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func TestPostRepository_RepliesAndReactions(t *testing.T) {
	db, cleanup := setupPostgres(t)
	defer cleanup()

	repo := NewPost(db, NewOutbox(db))
	post := &domain.Post{ID: uuid.New(), AuthorID: uuid.New(), Pseudonym: "post-author", Content: "foo"}
	if err := repo.Create(context.Background(), post); err != nil {
		t.Fatalf("create post failed: %v", err)
	}

	reply := &domain.Reply{PostID: post.ID, AuthorID: uuid.New(), Pseudonym: "reply-user", Content: "nice"}
	if err := repo.CreateReply(context.Background(), reply); err != nil {
		t.Fatalf("create reply failed: %v", err)
	}

	items, err := repo.ListReplies(context.Background(), post.ID, 10)
	if err != nil {
		t.Fatalf("list replies failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 reply got %d", len(items))
	}

	if err := repo.UpsertReaction(context.Background(), post.ID, reply.AuthorID, domain.Upvote); err != nil {
		t.Fatalf("upsert reaction failed: %v", err)
	}

	p2, err := repo.GetByID(context.Background(), post.ID)
	if err != nil {
		t.Fatalf("get post failed: %v", err)
	}
	if p2.Score != 1 {
		t.Fatalf("expected score 1 got %d", p2.Score)
	}

	if err := repo.UpsertReaction(context.Background(), post.ID, reply.AuthorID, domain.Upvote); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict on duplicate upvote, got %v", err)
	}

	p3, err := repo.GetByID(context.Background(), post.ID)
	if err != nil {
		t.Fatalf("get post failed: %v", err)
	}
	if p3.Score != 1 {
		t.Fatalf("expected score to remain 1 got %d", p3.Score)
	}
}
