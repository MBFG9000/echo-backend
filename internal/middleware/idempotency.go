package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const idempotencyTTL = 24 * time.Hour

type idempotencyRecord struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    json.RawMessage   `json:"body"`
}

type responseCapture struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseCapture) Write(b []byte) (int, error) {
	if w.body != nil {
		_, _ = w.body.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func (w *responseCapture) WriteString(s string) (int, error) {
	if w.body != nil {
		_, _ = w.body.WriteString(s)
	}
	return w.ResponseWriter.WriteString(s)
}

func Idempotency(redisClient *redis.Client) gin.HandlerFunc {
	store := &idempotencyStore{redis: redisClient}
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}

		scope, extra := idempotencyScope(c)
		if scope == "" {
			c.Next()
			return
		}

		key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
		if key == "" {
			c.Next()
			return
		}

		if _, err := uuid.Parse(key); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid idempotency key", "code": "ERR_INVALID_INPUT"})
			c.Abort()
			return
		}

		subject := idempotencySubject(c)
		cacheKey := "idempotency:" + scope + ":" + subject + ":" + extra + ":" + key

		if stored, ok := store.get(c.Request.Context(), cacheKey); ok {
			replay(c, stored)
			c.Abort()
			return
		}

		capture := &responseCapture{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = capture
		c.Next()

		if capture.body.Len() == 0 {
			return
		}

		status := c.Writer.Status()
		if status < 200 || status >= 300 {
			return
		}

		record := idempotencyRecord{
			Status: status,
			Body:   append(json.RawMessage(nil), capture.body.Bytes()...),
		}
		if contentType := c.Writer.Header().Get("Content-Type"); contentType != "" {
			record.Headers = map[string]string{"Content-Type": contentType}
		}
		_ = store.save(c.Request.Context(), cacheKey, record)
	}
}

func idempotencyScope(c *gin.Context) (scope, extra string) {
	path := c.FullPath()
	switch {
	case path == "/posts" && c.Request.Method == http.MethodPost:
		return "post", ""
	case path == "/posts/:id/replies":
		return "reply", c.Param("id")
	case path == "/posts/replies/create":
		return "reply", ""
	default:
		return "", ""
	}
}

func idempotencySubject(c *gin.Context) string {
	if v, ok := c.Get("userID"); ok {
		if id, ok := v.(uuid.UUID); ok {
			return id.String()
		}
	}
	return "anon"
}

type idempotencyStore struct {
	redis *redis.Client
}

func (s *idempotencyStore) get(ctx context.Context, key string) (idempotencyRecord, bool) {
	raw, err := s.redis.Get(ctx, key).Bytes()
	if err != nil {
		return idempotencyRecord{}, false
	}

	var record idempotencyRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return idempotencyRecord{}, false
	}

	return record, true
}

func (s *idempotencyStore) save(ctx context.Context, key string, record idempotencyRecord) error {
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}

	return s.redis.Set(ctx, key, raw, idempotencyTTL).Err()
}

func replay(c *gin.Context, record idempotencyRecord) {
	for k, v := range record.Headers {
		c.Writer.Header().Set(k, v)
	}
	c.Status(record.Status)
	_, _ = c.Writer.Write(record.Body)
}
