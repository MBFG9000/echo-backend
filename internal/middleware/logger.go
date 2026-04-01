package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

type Logger struct {
	logger *slog.Logger
}

func NewLogger(logger *slog.Logger) *Logger {
	return &Logger{logger: logger}
}

func (l *Logger) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		attrs := []any{
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", time.Since(startedAt)),
		}

		if v, ok := c.Get("pseudonym"); ok {
			if s, ok := v.(string); ok && s != "" {
				attrs = append(attrs, slog.String("pseudonym", s))
			}
		}

		l.logger.Info("http_request", attrs...)
	}
}
