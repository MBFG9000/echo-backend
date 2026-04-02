package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/echo-app/echo/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type postSvcStub struct {
	create  func(ctx context.Context, authorID uuid.UUID, pseudonym, content string) (*domain.Post, error)
	delete  func(ctx context.Context, postID, authorID uuid.UUID) error
	getByID func(ctx context.Context, postID uuid.UUID) (*domain.Post, error)
}

func (s *postSvcStub) Create(ctx context.Context, authorID uuid.UUID, pseudonym, content string) (*domain.Post, error) {
	return s.create(ctx, authorID, pseudonym, content)
}

func (s *postSvcStub) Delete(ctx context.Context, postID, authorID uuid.UUID) error {
	return s.delete(ctx, postID, authorID)
}

func (s *postSvcStub) GetByID(ctx context.Context, postID uuid.UUID) (*domain.Post, error) {
	return s.getByID(ctx, postID)
}

func (s *postSvcStub) React(ctx context.Context, postID, userID uuid.UUID, kind domain.ReactionKind) error {
	return nil
}

func (s *postSvcStub) CreateReply(ctx context.Context, postID, authorID uuid.UUID, pseudonym, content string) (*domain.Reply, error) {
	return nil, nil
}

func (s *postSvcStub) ListReplies(ctx context.Context, postID uuid.UUID, limit int) ([]domain.Reply, error) {
	return nil, nil
}

func TestPostHandler_Create(t *testing.T) {
	cases := []struct {
		name           string
		payload        interface{}
		userID         uuid.UUID
		pseudonym      string
		service        *postSvcStub
		expectedStatus int
	}{
		{name: "json invalid", payload: "{", userID: uuid.New(), pseudonym: "p", service: &postSvcStub{}, expectedStatus: http.StatusBadRequest},
		{name: "unauthorized", payload: map[string]string{"content": "hello"}, userID: uuid.Nil, pseudonym: "", service: &postSvcStub{}, expectedStatus: http.StatusUnauthorized},
		{name: "success", payload: map[string]string{"content": "hello"}, userID: uuid.New(), pseudonym: "echo", service: &postSvcStub{create: func(ctx context.Context, authorID uuid.UUID, pseudonym, content string) (*domain.Post, error) {
			return &domain.Post{ID: uuid.New()}, nil
		}}, expectedStatus: http.StatusCreated},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			p := NewPost(tc.service)
			p.RegisterPrivate(r.Group("/posts"), func(c *gin.Context) {
				if tc.userID != uuid.Nil {
					c.Set("userID", tc.userID)
					c.Set("pseudonym", tc.pseudonym)
				}
				c.Next()
			})

			b, _ := json.Marshal(tc.payload)
			req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewBuffer(b))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Fatalf("expected %d got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}

func TestPostHandler_GetByID(t *testing.T) {
	postID := uuid.New()
	cases := []struct {
		name           string
		id             string
		service        *postSvcStub
		expectedStatus int
	}{
		{name: "invalid id", id: "bad", service: &postSvcStub{}, expectedStatus: http.StatusBadRequest},
		{name: "not found", id: postID.String(), service: &postSvcStub{getByID: func(ctx context.Context, id uuid.UUID) (*domain.Post, error) { return nil, domain.ErrNotFound }}, expectedStatus: http.StatusNotFound},
		{name: "success", id: postID.String(), service: &postSvcStub{getByID: func(ctx context.Context, id uuid.UUID) (*domain.Post, error) { return &domain.Post{ID: postID}, nil }}, expectedStatus: http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			p := NewPost(tc.service)
			p.RegisterPublic(r.Group("/posts"))

			req := httptest.NewRequest(http.MethodGet, "/posts/"+tc.id, nil)
			body, _ := json.Marshal(map[string]string{"id": tc.id})
			req = httptest.NewRequest(http.MethodPost, "/posts/get", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Fatalf("expected %d got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}

func TestPostHandler_Delete(t *testing.T) {
	postID := uuid.New()
	ownerID := uuid.New()
	cases := []struct {
		name           string
		userID         uuid.UUID
		service        *postSvcStub
		expectedStatus int
	}{
		{name: "unauthorized", userID: uuid.Nil, service: &postSvcStub{}, expectedStatus: http.StatusUnauthorized},
		{name: "not found", userID: ownerID, service: &postSvcStub{delete: func(ctx context.Context, postID, authorID uuid.UUID) error { return domain.ErrNotFound }}, expectedStatus: http.StatusNotFound},
		{name: "success", userID: ownerID, service: &postSvcStub{delete: func(ctx context.Context, postID, authorID uuid.UUID) error { return nil }}, expectedStatus: http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.Use(func(c *gin.Context) {
				if tc.userID != uuid.Nil {
					c.Set("userID", tc.userID)
				}
				c.Next()
			})
			p := NewPost(tc.service)
			p.RegisterPrivate(r.Group("/posts"))

			body, _ := json.Marshal(map[string]string{"id": postID.String()})
			req := httptest.NewRequest(http.MethodPost, "/posts/delete", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Fatalf("expected %d got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}
