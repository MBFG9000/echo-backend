package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/echo-app/echo/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Post struct {
	posts        domain.PostService
	publicAppURL string
}

type createPostRequest struct {
	Content string `json:"content" binding:"required,max=280"`
}

type getPostRequest struct {
	ID string `json:"id" binding:"required"`
}

type deletePostRequest struct {
	ID string `json:"id" binding:"required"`
}

type shareResponse struct {
	URL    string `json:"url"`
	PostID string `json:"postId"`
}

type searchPostsRequest struct {
	Query string `json:"query" binding:"required"`
	Limit int    `json:"limit"`
}

func NewPost(posts domain.PostService, publicAppURL string) *Post {
	return &Post{
		posts:        posts,
		publicAppURL: strings.TrimRight(strings.TrimSpace(publicAppURL), "/"),
	}
}

func (p *Post) RegisterPublic(rg *gin.RouterGroup) {
	rg.POST("/get", p.getByID)
	rg.POST("/search", p.search)
	rg.GET("/:id/share", p.share)
	rg.GET("/:id", p.getByIDFromParam)
}

func (p *Post) RegisterPrivate(rg *gin.RouterGroup, createMiddleware ...gin.HandlerFunc) {
	if len(createMiddleware) == 0 {
		rg.POST("", p.create)
	} else {
		handlers := make([]gin.HandlerFunc, 0, len(createMiddleware)+1)
		handlers = append(handlers, createMiddleware...)
		handlers = append(handlers, p.create)
		rg.POST("", handlers...)
	}

	rg.POST("/delete", p.delete)
	rg.DELETE("/:id", p.deleteFromParam)
}

func (p *Post) create(c *gin.Context) {
	var req createPostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	userIDValue, ok := c.Get("userID")
	if !ok {
		writeDomainError(c, domain.ErrUnauthorized)
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		writeDomainError(c, domain.ErrUnauthorized)
		return
	}

	pseudonymValue, ok := c.Get("pseudonym")
	if !ok {
		writeDomainError(c, domain.ErrUnauthorized)
		return
	}

	pseudonym, ok := pseudonymValue.(string)
	if !ok {
		writeDomainError(c, domain.ErrUnauthorized)
		return
	}

	post, err := p.posts.Create(c.Request.Context(), userID, pseudonym, req.Content)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, post)
}

func (p *Post) delete(c *gin.Context) {
	var req deletePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	p.deletePost(c, req.ID)
}

func (p *Post) deleteFromParam(c *gin.Context) {
	p.deletePost(c, c.Param("id"))
}

func (p *Post) deletePost(c *gin.Context, rawID string) {
	userIDValue, ok := c.Get("userID")
	if !ok {
		writeDomainError(c, domain.ErrUnauthorized)
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		writeDomainError(c, domain.ErrUnauthorized)
		return
	}

	postID, err := uuid.Parse(rawID)
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	err = p.posts.Delete(c.Request.Context(), postID, userID)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (p *Post) share(c *gin.Context) {
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	if _, err := p.posts.GetByID(c.Request.Context(), postID); err != nil {
		writeDomainError(c, err)
		return
	}

	if p.publicAppURL == "" {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	c.JSON(http.StatusOK, shareResponse{
		URL:    fmt.Sprintf("%s/post/%s", p.publicAppURL, postID.String()),
		PostID: postID.String(),
	})
}

func (p *Post) getByID(c *gin.Context) {
	var req getPostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	p.respondPost(c, req.ID)
}

func (p *Post) getByIDFromParam(c *gin.Context) {
	p.respondPost(c, c.Param("id"))
}

func (p *Post) respondPost(c *gin.Context, rawID string) {
	postID, err := uuid.Parse(rawID)
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	post, err := p.posts.GetByID(c.Request.Context(), postID)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, post)
}

func (p *Post) search(c *gin.Context) {
	var req searchPostsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	posts, err := p.posts.Search(c.Request.Context(), req.Query, limit)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"posts": posts})
}
