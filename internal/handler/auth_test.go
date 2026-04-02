package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/echo-app/echo/internal/domain"
	"github.com/gin-gonic/gin"
)

type authSvcStub struct {
	register func(ctx context.Context) (string, string, error)
	refresh  func(ctx context.Context, token string) (string, error)
}

func (s *authSvcStub) Register(ctx context.Context) (string, string, error) {
	return s.register(ctx)
}

func (s *authSvcStub) Refresh(ctx context.Context, token string) (string, error) {
	return s.refresh(ctx, token)
}

func TestAuthHandler_Register(t *testing.T) {
	cases := []struct {
		name           string
		svc            *authSvcStub
		expectedStatus int
		expectedCode   string
	}{
		{name: "success", svc: &authSvcStub{register: func(ctx context.Context) (string, string, error) { return "tok", "pseudo", nil }}, expectedStatus: http.StatusCreated},
		{name: "domain conflict", svc: &authSvcStub{register: func(ctx context.Context) (string, string, error) { return "", "", domain.ErrConflict }}, expectedStatus: http.StatusConflict, expectedCode: "ERR_CONFLICT"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			h := NewAuth(tc.svc)
			h.Register(r.Group("/auth"))

			req := httptest.NewRequest(http.MethodPost, "/auth/register", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Fatalf("expected status %d got %d", tc.expectedStatus, w.Code)
			}

			if tc.expectedCode != "" {
				var body map[string]string
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body["code"] != tc.expectedCode {
					t.Fatalf("expected code %s got %s", tc.expectedCode, body["code"])
				}
			}
		})
	}
}

func TestAuthHandler_Refresh(t *testing.T) {
	cases := []struct {
		name           string
		headers        map[string]string
		svc            *authSvcStub
		expectedStatus int
		expectedCode   string
	}{
		{name: "missing auth", headers: nil, expectedStatus: http.StatusUnauthorized, expectedCode: "ERR_UNAUTHORIZED"},
		{name: "bad format", headers: map[string]string{"Authorization": "Bad token"}, expectedStatus: http.StatusUnauthorized, expectedCode: "ERR_UNAUTHORIZED"},
		{name: "success", headers: map[string]string{"Authorization": "Bearer tok"}, svc: &authSvcStub{refresh: func(ctx context.Context, token string) (string, error) { return "newtok", nil }}, expectedStatus: http.StatusOK},
		{name: "unauthorized", headers: map[string]string{"Authorization": "Bearer tok"}, svc: &authSvcStub{refresh: func(ctx context.Context, token string) (string, error) { return "", domain.ErrUnauthorized }}, expectedStatus: http.StatusUnauthorized, expectedCode: "ERR_UNAUTHORIZED"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			h := NewAuth(tc.svc)
			h.Register(r.Group("/auth"))

			req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Fatalf("expected status %d got %d", tc.expectedStatus, w.Code)
			}

			if tc.expectedCode != "" {
				var body map[string]string
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body["code"] != tc.expectedCode {
					t.Fatalf("expected code %s got %s", tc.expectedCode, body["code"])
				}
			}
		})
	}
}
