package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/echo-app/echo/internal/domain"
	"github.com/echo-app/echo/internal/middleware"
	"github.com/echo-app/echo/internal/service"
	"github.com/gin-gonic/gin"
)

type Feed struct {
	feeds domain.FeedService
	posts domain.PostService
	auth  *middleware.Auth
}

type latestFeedRequest struct {
	Limit  int    `json:"limit"`
	Cursor string `json:"cursor"`
}

type trendingFeedRequest struct {
	Limit int `json:"limit"`
}

type latestFeedResponse struct {
	Posts      []domain.Post `json:"posts"`
	NextCursor string        `json:"next_cursor"`
}

type trendingFeedResponse struct {
	Posts []domain.Post `json:"posts"`
}

func NewFeed(feeds domain.FeedService, posts domain.PostService, auth *middleware.Auth) *Feed {
	return &Feed{feeds: feeds, posts: posts, auth: auth}
}

func (f *Feed) Register(rg *gin.RouterGroup) {
	rg.GET("/latest", f.latest)
	rg.POST("/latest", f.latest)
	rg.GET("/trending", f.trending)
	rg.POST("/trending", f.trending)
}

// @Summary Latest feed
// @Tags feed
// @Accept json
// @Produce json
// @Param request body latestFeedRequest true "Feed payload"
// @Success 200 {object} latestFeedResponse
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /feed/latest [post]
func (f *Feed) latest(c *gin.Context) {
	limit, cursorRaw, err := f.latestInput(c)
	if err != nil {
		if c.Request.Method == http.MethodPost {
			writeValidationError(c, err)
		} else {
			writeDomainError(c, domain.ErrInvalidInput)
		}
		return
	}

	cursor, err := parseCursor(cursorRaw)
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	posts, next, err := f.feeds.Latest(c.Request.Context(), limit, cursor)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	if f.auth != nil {
		if userID, ok := f.auth.TryUserID(c); ok {
			f.posts.MarkViewerReactionsOnPosts(c.Request.Context(), userID, posts)
		}
	}

	nextValue := ""
	if next != nil {
		nextValue = next.CreatedAt.Format("2006-01-02T15:04:05.999999999Z07:00")
	}

	if c.Request.Method == http.MethodGet {
		etag := service.LatestFeedETag(limit, cursorRaw, posts)
		c.Header("ETag", etag)
		if inm := strings.TrimSpace(c.GetHeader("If-None-Match")); inm == etag {
			c.Status(http.StatusNotModified)
			return
		}
	}

	c.JSON(http.StatusOK, latestFeedResponse{Posts: posts, NextCursor: nextValue})
}

// @Summary Trending feed
// @Tags feed
// @Accept json
// @Produce json
// @Param request body trendingFeedRequest true "Feed payload"
// @Success 200 {object} trendingFeedResponse
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /feed/trending [post]
func (f *Feed) trending(c *gin.Context) {
	limit, err := f.trendingLimit(c)
	if err != nil {
		if c.Request.Method == http.MethodPost {
			writeValidationError(c, err)
		} else {
			writeDomainError(c, domain.ErrInvalidInput)
		}
		return
	}

	posts, err := f.feeds.Trending(c.Request.Context(), limit)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	if f.auth != nil {
		if userID, ok := f.auth.TryUserID(c); ok {
			f.posts.MarkViewerReactionsOnPosts(c.Request.Context(), userID, posts)
		}
	}

	c.JSON(http.StatusOK, trendingFeedResponse{Posts: posts})
}

func (f *Feed) latestInput(c *gin.Context) (int, string, error) {
	if c.Request.Method == http.MethodGet {
		limit := 20
		if raw := c.Query("limit"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				return 0, "", err
			}
			limit = parsed
		}

		return limit, c.Query("cursor"), nil
	}

	var req latestFeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return 0, "", err
	}

	limit := 20
	if req.Limit > 0 {
		limit = req.Limit
	}

	return limit, req.Cursor, nil
}

func (f *Feed) trendingLimit(c *gin.Context) (int, error) {
	if c.Request.Method == http.MethodGet {
		limit := 20
		if raw := c.Query("limit"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				return 0, err
			}
			limit = parsed
		}

		return limit, nil
	}

	var req trendingFeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return 0, err
	}

	limit := 20
	if req.Limit > 0 {
		limit = req.Limit
	}

	return limit, nil
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
