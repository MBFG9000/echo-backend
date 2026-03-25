package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

type RequestLog struct {
	logger *slog.Logger
}

func NewRequestLog(logger *slog.Logger) *RequestLog {
	return &RequestLog{logger: logger}
}

func (r *RequestLog) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		r.logger.Info("http_request",
			slog.String("method", c.Request.Method),
			slog.String("path", c.FullPath()),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", time.Since(startedAt)),
		)
	}
}
