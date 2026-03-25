package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type CORS struct {
	allowAll bool
	origins  map[string]struct{}
}

func NewCORS(allowedOrigins []string) *CORS {
	origins := make(map[string]struct{}, len(allowedOrigins))
	allowAll := false

	for _, origin := range allowedOrigins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		if trimmed == "*" {
			allowAll = true
			continue
		}
		origins[trimmed] = struct{}{}
	}

	return &CORS{allowAll: allowAll, origins: origins}
}

func (m *CORS) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		if m.allowAll {
			c.Header("Access-Control-Allow-Origin", "*")
		} else {
			if _, ok := m.origins[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")

		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}

		c.Next()
	}
}
