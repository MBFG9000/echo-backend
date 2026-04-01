package middleware

import (
	"github.com/echo-app/echo/internal/audit"
	"github.com/gin-gonic/gin"
)

var stripHeaderNames = []string{
	"X-Forwarded-For",
	"X-Real-Ip",
	"Cf-Connecting-Ip",
	"True-Client-Ip",
}

func NoIP(development bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, name := range stripHeaderNames {
			c.Request.Header.Del(name)
		}
		c.Set("client_ip", "")
		audit.NoIPContext(c, development)
		c.Next()
	}
}
