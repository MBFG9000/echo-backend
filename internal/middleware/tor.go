package middleware

import (
	"strings"

	"github.com/echo-app/echo/internal/config"
	"github.com/gin-gonic/gin"
)

func Tor(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Server", "")
		h.Del("X-Powered-By")

		onion := strings.TrimSpace(cfg.OnionAddress)
		if onion != "" {
			onion = strings.TrimPrefix(onion, "http://")
			onion = strings.TrimPrefix(onion, "https://")
			onion = strings.TrimSuffix(onion, "/")
			h.Set("Onion-Location", "http://"+onion+c.Request.URL.RequestURI())
		}

		c.Next()
	}
}
