package audit

import (
	"github.com/gin-gonic/gin"
)

func NoIPContext(c *gin.Context, development bool) {
	if !development {
		return
	}
	if c.Request.Header.Get("X-Forwarded-For") != "" || c.Request.Header.Get("X-Real-Ip") != "" {
		panic("ip leak: forwarded headers still present after NoIP")
	}
}
