package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/echo-app/echo/internal/domain"
)

func LatestFeedETag(limit int, cursorRaw string, posts []domain.Post) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "limit=%d&cursor=%s", limit, cursorRaw)
	for _, post := range posts {
		_, _ = fmt.Fprintf(h, "|%s:%d:%d", post.ID, post.CreatedAt.UnixNano(), post.Score)
	}
	return `W/"` + hex.EncodeToString(h.Sum(nil)[:16]) + `"`
}
