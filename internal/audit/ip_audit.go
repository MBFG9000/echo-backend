package audit

import (
	"github.com/gin-gonic/gin"
)

var ipHeaders = []string{
	"X-Forwarded-For",
	"X-Real-Ip",
	"Cf-Connecting-Ip",
	"True-Client-Ip",
}

func NoIPContext(c *gin.Context, development bool) {
	if !development {
		return
	}

	for _, header := range ipHeaders {
		if c.Request.Header.Get(header) != "" {
			panic("ip leak: forwarded headers still present after NoIP")
		}
	}
}
