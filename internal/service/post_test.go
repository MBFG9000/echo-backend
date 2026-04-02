package service

import (
	"context"
	"errors"
	"testing"

	"github.com/echo-app/echo/internal/domain"
	"github.com/google/uuid"
)

type postRepoStub struct {
	create       func(ctx context.Context, post *domain.Post) error
	deleteByAuth func(ctx context.Context, postID, authorID uuid.UUID) error
	getByID      func(ctx context.Context, postID uuid.UUID) (*domain.Post, error)
	setHidden    func(ctx context.Context, postID uuid.UUID, hidden bool) error
	createReply  func(ctx context.Context, reply *domain.Reply) error
	listReplies  func(ctx context.Context, postID uuid.UUID, limit int) ([]domain.Reply, error)
	upsertReact  func(ctx context.Context, postID, userID uuid.UUID, kind domain.ReactionKind) error
}

func (s *postRepoStub) Create(ctx context.Context, post *domain.Post) error {
	if s.create != nil {
		return s.create(ctx, post)
	}
	return nil
}

func (s *postRepoStub) DeleteByAuthor(ctx context.Context, postID, authorID uuid.UUID) error {
	if s.deleteByAuth != nil {
		return s.deleteByAuth(ctx, postID, authorID)
	}
	return nil
}

func (s *postRepoStub) GetByID(ctx context.Context, postID uuid.UUID) (*domain.Post, error) {
	if s.getByID != nil {
		return s.getByID(ctx, postID)
	}
	return nil, domain.ErrNotFound
}

func (s *postRepoStub) SetHidden(ctx context.Context, postID uuid.UUID, hidden bool) error {
	if s.setHidden != nil {
		return s.setHidden(ctx, postID, hidden)
	}
	return nil
}

func (s *postRepoStub) CreateReply(ctx context.Context, reply *domain.Reply) error {
	if s.createReply != nil {
		return s.createReply(ctx, reply)
	}
	return nil
}

func (s *postRepoStub) ListReplies(ctx context.Context, postID uuid.UUID, limit int) ([]domain.Reply, error) {
	if s.listReplies != nil {
		return s.listReplies(ctx, postID, limit)
	}
	return nil, nil
}

func (s *postRepoStub) UpsertReaction(ctx context.Context, postID, userID uuid.UUID, kind domain.ReactionKind) error {
	if s.upsertReact != nil {
		return s.upsertReact(ctx, postID, userID, kind)
	}
	return nil
}

type broadcasterStub struct {
	payload []byte
}

func (b *broadcasterStub) Broadcast(payload []byte) {
	b.payload = payload
}

func TestPost_Create(t *testing.T) {
	created := false
	broadcaster := &broadcasterStub{}
	stubRepo := &postRepoStub{}
	stubRepo.create = func(ctx context.Context, post *domain.Post) error {
		created = true
		return nil
	}

	p := NewPost(stubRepo, broadcaster)

	actionables := []struct {
		name      string
		pseudonym string
		content   string
		wantErr   error
	}{
		{name: "valid input", pseudonym: "echo-hero", content: "hello world", wantErr: nil},
		{name: "empty content", pseudonym: "echo-hero", content: " ", wantErr: domain.ErrInvalidInput},
		{name: "empty pseudonym", pseudonym: " ", content: "hello", wantErr: domain.ErrInvalidInput},
	}

	for _, tc := range actionables {
		t.Run(tc.name, func(t *testing.T) {
			created = false
			broadcaster.payload = nil
			post, err := p.Create(context.Background(), uuid.New(), tc.pseudonym, tc.content)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v got %v", tc.wantErr, err)
			}
			if tc.wantErr == nil {
				if post == nil {
					t.Fatal("expected post")
				}
				if !created {
					t.Fatal("expected Create() called")
				}
				if broadcaster.payload == nil {
					t.Fatal("expected broadcast payload")
				}
			}
		})
	}
}

func TestPost_Delete(t *testing.T) {
	postID := uuid.New()
	authorID := uuid.New()
	otherID := uuid.New()

	cases := []struct {
		name     string
		post     *domain.Post
		userID   uuid.UUID
		wantErr  error
		deleteFn func(ctx context.Context, postID, authorID uuid.UUID) error
	}{
		{name: "success", post: &domain.Post{ID: postID, AuthorID: authorID}, userID: authorID, wantErr: nil, deleteFn: func(ctx context.Context, postID, authorID uuid.UUID) error { return nil }},
		{name: "not owner", post: &domain.Post{ID: postID, AuthorID: authorID}, userID: otherID, wantErr: domain.ErrUnauthorized, deleteFn: nil},
		{name: "not found", post: nil, userID: authorID, wantErr: domain.ErrNotFound, deleteFn: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stubRepo := &postRepoStub{}
			stubRepo.getByID = func(ctx context.Context, id uuid.UUID) (*domain.Post, error) {
				if tc.post == nil {
					return nil, domain.ErrNotFound
				}
				return tc.post, nil
			}
			stubRepo.deleteByAuth = func(ctx context.Context, postID, authorID uuid.UUID) error {
				if tc.deleteFn != nil {
					return tc.deleteFn(ctx, postID, authorID)
				}
				return nil
			}

			p := NewPost(stubRepo, nil)
			err := p.Delete(context.Background(), postID, tc.userID)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v got %v", tc.wantErr, err)
			}
		})
	}
}

func TestPost_React(t *testing.T) {
	postID := uuid.New()
	userID := uuid.New()

	cases := []struct {
		name    string
		kind    domain.ReactionKind
		postOK  bool
		wantErr error
	}{
		{name: "invalid kind", kind: "invalid", postOK: true, wantErr: domain.ErrInvalidInput},
		{name: "post not found", kind: domain.Upvote, postOK: false, wantErr: domain.ErrNotFound},
		{name: "success", kind: domain.Downvote, postOK: true, wantErr: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stubRepo := &postRepoStub{}
			stubRepo.getByID = func(ctx context.Context, id uuid.UUID) (*domain.Post, error) {
				if !tc.postOK {
					return nil, domain.ErrNotFound
				}
				return &domain.Post{ID: postID}, nil
			}
			stubRepo.upsertReact = func(ctx context.Context, postID, userID uuid.UUID, kind domain.ReactionKind) error {
				return nil
			}

			p := NewPost(stubRepo, nil)
			err := p.React(context.Background(), postID, userID, tc.kind)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v got %v", tc.wantErr, err)
			}
		})
	}
}

func TestPost_CreateReplyAndList(t *testing.T) {
	postID := uuid.New()
	authorID := uuid.New()

	stubRepo := &postRepoStub{}
	stubRepo.getByID = func(ctx context.Context, id uuid.UUID) (*domain.Post, error) {
		if id != postID {
			return nil, domain.ErrNotFound
		}
		return &domain.Post{ID: postID}, nil
	}
	stubRepo.createReply = func(ctx context.Context, reply *domain.Reply) error {
		return nil
	}
	stubRepo.listReplies = func(ctx context.Context, id uuid.UUID, limit int) ([]domain.Reply, error) {
		return []domain.Reply{{ID: uuid.New(), PostID: postID, AuthorID: authorID, Content: "a"}}, nil
	}

	p := NewPost(stubRepo, nil)
	_, err := p.CreateReply(context.Background(), postID, authorID, "pseudonym", "reply")
	if err != nil {
		t.Fatal(err)
	}

	replies, err := p.ListReplies(context.Background(), postID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(replies) != 1 {
		t.Fatalf("expected 1 reply got %d", len(replies))
	}
}
