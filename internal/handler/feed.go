package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/echo-app/echo/internal/domain"
	"github.com/gin-gonic/gin"
)

type Feed struct {
	feeds domain.FeedService
}

type latestFeedRequest struct {
	Limit  int    `json:"limit"`
	Cursor string `json:"cursor"`
}

type trendingFeedRequest struct {
	Limit int `json:"limit"`
}

func NewFeed(feeds domain.FeedService) *Feed {
	return &Feed{feeds: feeds}
}

func (f *Feed) Register(rg *gin.RouterGroup) {
	rg.POST("/latest", f.latest)
	rg.POST("/trending", f.trending)
}

func (f *Feed) latest(c *gin.Context) {
	var req latestFeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	limit := 20
	if req.Limit > 0 {
		limit = req.Limit
	}

	cursor, err := parseCursor(req.Cursor)
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	posts, next, err := f.feeds.Latest(c.Request.Context(), limit, cursor)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	nextValue := ""
	if next != nil {
		nextValue = next.CreatedAt.Format("2006-01-02T15:04:05.999999999Z07:00")
	}

	c.JSON(http.StatusOK, gin.H{"posts": posts, "next_cursor": nextValue})
}

func (f *Feed) trending(c *gin.Context) {
	var req trendingFeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	limit := 20
	if req.Limit > 0 {
		limit = req.Limit
	}

	posts, err := f.feeds.Trending(c.Request.Context(), limit)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"posts": posts})
}

func parseCursor(raw string) (*domain.FeedCursor, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	createdAt, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	return &domain.FeedCursor{CreatedAt: createdAt}, nil
}
