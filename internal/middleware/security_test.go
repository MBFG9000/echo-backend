package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/echo-app/echo/internal/config"
	"github.com/echo-app/echo/internal/middleware"
	"github.com/gin-gonic/gin"
)

func TestSecurityHeadersAndNoIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	devCfg := config.Config{Env: "development"}
	prodCfg := config.Config{Env: "production"}

	tests := []struct {
		name       string
		cfg        config.Config
		req        func() *http.Request
		assertBody bool
		check      func(t *testing.T, rec *httptest.ResponseRecorder, body []byte)
	}{
		{
			name: "csp_on_response",
			cfg:  devCfg,
			req: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/x", nil)
			},
			check: func(t *testing.T, rec *httptest.ResponseRecorder, _ []byte) {
				t.Helper()
				if got := rec.Header().Get("Content-Security-Policy"); got != "default-src 'self'" {
					t.Fatalf("CSP got %q", got)
				}
			},
		},
		{
			name: "xff_stripped_before_handler",
			cfg:  devCfg,
			req: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/xff", nil)
				r.Header.Set("X-Forwarded-For", "203.0.113.1")
				return r
			},
			assertBody: true,
			check: func(t *testing.T, rec *httptest.ResponseRecorder, body []byte) {
				t.Helper()
				var out struct {
					XFF string `json:"xff"`
				}
				if err := json.Unmarshal(body, &out); err != nil {
					t.Fatal(err)
				}
				if out.XFF != "" {
					t.Fatalf("handler saw XFF %q", out.XFF)
				}
			},
		},
		{
			name: "production_http_redirects_https",
			cfg:  prodCfg,
			req: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/p?q=1", nil)
				r.Host = "example.com"
				r.Header.Set("X-Forwarded-Proto", "http")
				return r
			},
			check: func(t *testing.T, rec *httptest.ResponseRecorder, _ []byte) {
				t.Helper()
				if rec.Code != http.StatusMovedPermanently {
					t.Fatalf("status %d", rec.Code)
				}
				loc := rec.Header().Get("Location")
				if !strings.HasPrefix(loc, "https://") {
					t.Fatalf("Location %q", loc)
				}
			},
		},
		{
			name: "server_header_not_gin",
			cfg:  devCfg,
			req: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/s", nil)
			},
			check: func(t *testing.T, rec *httptest.ResponseRecorder, _ []byte) {
				t.Helper()
				if rec.Header().Get("Server") == "gin" {
					t.Fatal("Server is gin")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(
				middleware.NoIP(true),
				middleware.Security(tt.cfg),
				middleware.Tor(tt.cfg),
			)
			r.GET("/xff", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"xff": c.GetHeader("X-Forwarded-For")})
			})
			r.GET("/x", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})
			r.GET("/p", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})
			r.GET("/s", func(c *gin.Context) {
				c.String(http.StatusOK, "ok")
			})

			req := tt.req()
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			var body []byte
			if tt.assertBody {
				body = rec.Body.Bytes()
			}
			tt.check(t, rec, body)
		})
	}
}

func TestLoggerNoRefererOrIPLeak(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	devCfg := config.Config{Env: "development"}

	r := gin.New()
	r.Use(
		middleware.NoIP(true),
		middleware.Security(devCfg),
		middleware.Tor(devCfg),
		middleware.NewLogger(log).Handler(),
	)
	r.GET("/z", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/z", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	lower := bytes.ToLower(buf.Bytes())
	if bytes.Contains(lower, []byte("referer")) {
		t.Fatal("log mentions referer")
	}
	if bytes.Contains(buf.Bytes(), []byte("203.0.113.1")) {
		t.Fatal("log contains client IP")
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("203.0.113.1")) {
		t.Fatal("response body contains IP")
	}
}
