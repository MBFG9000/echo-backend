package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDKey = "requestID"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		c.Set(RequestIDKey, id)
		c.Writer.Header().Set("X-Request-ID", id)
		c.Next()
	}
}
