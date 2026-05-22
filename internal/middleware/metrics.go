package middleware

import (
	"strconv"
	"time"

	"github.com/echo-app/echo/internal/metrics"
	"github.com/gin-gonic/gin"
)

func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		metrics.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(method, path).Observe(time.Since(startedAt).Seconds())
	}
}
