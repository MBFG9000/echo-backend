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

func (s *authSvcStub) AdminLogin(ctx context.Context, username, password string) (string, error) {
	return "", domain.ErrUnauthorized
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
				var body struct {
					Code string `json:"code"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Code != tc.expectedCode {
					t.Fatalf("expected code %s got %s", tc.expectedCode, body.Code)
				}
			}
		})
	}
}

func TestAuthHandler_Refresh(t *testing.T) {
	cases := []struct {
		name           string
		payload        string
		svc            *authSvcStub
		expectedStatus int
		expectedCode   string
	}{
		{name: "missing token", payload: `{}`, expectedStatus: http.StatusBadRequest, expectedCode: "ERR_VALIDATION"},
		{name: "success", payload: `{"token":"tok"}`, svc: &authSvcStub{refresh: func(ctx context.Context, token string) (string, error) { return "newtok", nil }}, expectedStatus: http.StatusOK},
		{name: "unauthorized", payload: `{"token":"tok"}`, svc: &authSvcStub{refresh: func(ctx context.Context, token string) (string, error) { return "", domain.ErrUnauthorized }}, expectedStatus: http.StatusUnauthorized, expectedCode: "ERR_UNAUTHORIZED"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			h := NewAuth(tc.svc)
			h.Register(r.Group("/auth"))

			req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString(tc.payload))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Fatalf("expected status %d got %d", tc.expectedStatus, w.Code)
			}

			if tc.expectedCode != "" {
				var body struct {
					Code string `json:"code"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Code != tc.expectedCode {
					t.Fatalf("expected code %s got %s", tc.expectedCode, body.Code)
				}
			}
		})
	}
}
