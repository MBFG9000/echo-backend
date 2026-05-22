package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/echo-app/echo/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Post struct {
	posts         domain.PostService
	publicAppURL  string
}

type createPostRequest struct {
	Content string `json:"content" binding:"required,max=280"`
}

func NewPost(posts domain.PostService, publicAppURL string) *Post {
	return &Post{
		posts:        posts,
		publicAppURL: strings.TrimRight(strings.TrimSpace(publicAppURL), "/"),
	}
}

type shareResponse struct {
	URL    string `json:"url"`
	PostID string `json:"postId"`
}

func (p *Post) RegisterPublic(rg *gin.RouterGroup) {
	rg.GET("/:id/share", p.share)
	rg.GET("/:id", p.getByID)
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

	rg.DELETE("/:id", p.delete)
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

	postID, err := uuid.Parse(c.Param("id"))
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
	postID, err := uuid.Parse(c.Param("id"))
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
