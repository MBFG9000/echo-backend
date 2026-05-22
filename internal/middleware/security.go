package middleware

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/echo-app/echo/internal/config"
	"github.com/gin-gonic/gin"
)

func Security(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.IsProduction() {
			proto := strings.ToLower(strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")))
			if proto != "https" {
				host := strings.TrimSpace(c.Request.Host)
				if host == "" {
					host = net.JoinHostPort(cfg.Server.Host, cfg.Server.Port)
				}
				target := url.URL{
					Scheme:   "https",
					Host:     host,
					Path:     c.Request.URL.Path,
					RawQuery: c.Request.URL.RawQuery,
				}
				setSecurityHeaders(c)
				c.Redirect(http.StatusMovedPermanently, target.String())
				c.Abort()
				return
			}
			c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		setSecurityHeaders(c)
		c.Next()
	}
}

func setSecurityHeaders(c *gin.Context) {
	h := c.Writer.Header()
	h.Set("Content-Security-Policy", "default-src 'self'")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
}
