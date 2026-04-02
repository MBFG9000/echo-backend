package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimit struct {
	redis            *redis.Client
	generalLimit     int64
	generalWindow    time.Duration
	postCreateLimit  int64
	postCreateWindow time.Duration
}

func NewRateLimit(redisClient *redis.Client, generalLimit int64, generalWindow time.Duration, postCreateLimit int64, postCreateWindow time.Duration) *RateLimit {
	if generalLimit <= 0 {
		generalLimit = 60
	}
	if generalWindow <= 0 {
		generalWindow = time.Minute
	}
	if postCreateLimit <= 0 {
		postCreateLimit = 10
	}
	if postCreateWindow <= 0 {
		postCreateWindow = time.Minute
	}

	return &RateLimit{
		redis:            redisClient,
		generalLimit:     generalLimit,
		generalWindow:    generalWindow,
		postCreateLimit:  postCreateLimit,
		postCreateWindow: postCreateWindow,
	}
}

func (r *RateLimit) General() gin.HandlerFunc {
	return r.handler("general", r.generalLimit, r.generalWindow)
}

func (r *RateLimit) PostCreate() gin.HandlerFunc {
	return r.handler("posts", r.postCreateLimit, r.postCreateWindow)
}

func (r *RateLimit) handler(scope string, limit int64, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		nowMillis := time.Now().UnixMilli()
		windowStart := nowMillis - window.Milliseconds()
		key := fmt.Sprintf("rate-limit:%s:%s", scope, subjectKey(c))
		member := strconv.FormatInt(time.Now().UnixNano(), 10)

		pipeline := r.redis.TxPipeline()
		pipeline.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(windowStart, 10))
		pipeline.ZAdd(ctx, key, redis.Z{Score: float64(nowMillis), Member: member})
		countCmd := pipeline.ZCard(ctx, key)
		pipeline.Expire(ctx, key, window+5*time.Second)
		_, err := pipeline.Exec(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, httpError{Error: "internal error", Code: "ERR_INTERNAL"})
			c.Abort()
			return
		}

		if countCmd.Val() > limit {
			c.Header("Retry-After", strconv.FormatInt(int64(window.Seconds()), 10))
			c.JSON(http.StatusTooManyRequests, httpError{Error: "rate limit exceeded", Code: "ERR_RATE_LIMIT"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func subjectKey(c *gin.Context) string {
	authorization := c.GetHeader("Authorization")
	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && strings.TrimSpace(parts[1]) != "" {
		sum := sha256.Sum256([]byte(parts[1]))
		return "token:" + hex.EncodeToString(sum[:])
	}

	return "subject:anon"
}
