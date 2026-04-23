package handler

import (
	"net/http"

	"github.com/echo-app/echo/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Post struct {
	posts domain.PostService
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

type searchPostsRequest struct {
	Query string `json:"query" binding:"required"`
	Limit int    `json:"limit"`
}

func NewPost(posts domain.PostService) *Post {
	return &Post{posts: posts}
}

func (p *Post) RegisterPublic(rg *gin.RouterGroup) {
	rg.POST("/get", p.getByID)
	rg.POST("/search", p.search)
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

	postID, err := uuid.Parse(req.ID)
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

func (p *Post) getByID(c *gin.Context) {
	var req getPostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	postID, err := uuid.Parse(req.ID)
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

	posts, err := p.posts.Search(c.Request.Context(), req.Query, req.Limit)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"posts": posts})
}
