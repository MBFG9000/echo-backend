package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/echo-app/echo/internal/domain"
	"github.com/gin-gonic/gin"
)

type Feed struct {
	feeds domain.FeedService
}

func NewFeed(feeds domain.FeedService) *Feed {
	return &Feed{feeds: feeds}
}

func (f *Feed) Register(rg *gin.RouterGroup) {
	rg.GET("/latest", f.latest)
	rg.GET("/trending", f.trending)
}

func (f *Feed) latest(c *gin.Context) {
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeDomainError(c, domain.ErrInvalidInput)
			return
		}
		limit = parsed
	}

	cursor, err := parseCursor(c.Query("cursor"))
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
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeDomainError(c, domain.ErrInvalidInput)
			return
		}
		limit = parsed
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
