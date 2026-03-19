package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/echo-app/echo/internal/domain"
	"github.com/gin-gonic/gin"
)

type Post struct {
	posts domain.PostService
}

type createPostRequest struct {
	Content string `json:"content" binding:"required,max=280"`
}

func NewPost(posts domain.PostService) *Post {
	return &Post{posts: posts}
}

func (p *Post) RegisterPublic(rg *gin.RouterGroup) {
	rg.GET("", p.listLatest)
}

func (p *Post) RegisterPrivate(rg *gin.RouterGroup) {
	rg.POST("", p.create)
}

func (p *Post) create(c *gin.Context) {
	var req createPostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}

	userIDValue, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrUnauthorized.Error()})
		return
	}

	userID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrUnauthorized.Error()})
		return
	}

	post, err := p.posts.Create(c.Request.Context(), userID, req.Content)
	if err != nil {
		p.writePostError(c, err)
		return
	}

	c.JSON(http.StatusCreated, post)
}

func (p *Post) listLatest(c *gin.Context) {
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
			return
		}
		limit = parsed
	}

	posts, err := p.posts.ListLatest(c.Request.Context(), limit)
	if err != nil {
		p.writePostError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"posts": posts})
}

func (p *Post) writePostError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrUnauthorized.Error()})
	case errors.Is(err, domain.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
