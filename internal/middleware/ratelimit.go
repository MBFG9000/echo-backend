package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimit struct {
	redis  *redis.Client
	limit  int64
	window time.Duration
}

func NewRateLimit(redisClient *redis.Client, limit int64, window time.Duration) *RateLimit {
	return &RateLimit{redis: redisClient, limit: limit, window: window}
}

func (r *RateLimit) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		key := "rate-limit:" + c.ClientIP()

		count, err := r.redis.Incr(ctx, key).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			c.Abort()
			return
		}

		if count == 1 {
			if err := r.redis.Expire(ctx, key, r.window).Err(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
				c.Abort()
				return
			}
		}

		if count > r.limit {
			c.Header("Retry-After", strconv.FormatInt(int64(r.window.Seconds()), 10))
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			c.Abort()
			return
		}

		c.Next()
	}
}
