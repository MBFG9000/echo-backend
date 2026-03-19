package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/echo-app/echo/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Reply struct {
	posts domain.PostService
}

type createReplyRequest struct {
	Content string `json:"content" binding:"required,max=280"`
}

func NewReply(posts domain.PostService) *Reply {
	return &Reply{posts: posts}
}

func (r *Reply) RegisterPublic(rg *gin.RouterGroup) {
	rg.GET("/:id/replies", r.list)
}

func (r *Reply) RegisterPrivate(rg *gin.RouterGroup) {
	rg.POST("/:id/replies", r.create)
}

func (r *Reply) create(c *gin.Context) {
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}

	var req createReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}

	userIDValue, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrUnauthorized.Error()})
		return
	}
	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrUnauthorized.Error()})
		return
	}

	pseudonymValue, ok := c.Get("pseudonym")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrUnauthorized.Error()})
		return
	}
	pseudonym, ok := pseudonymValue.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrUnauthorized.Error()})
		return
	}

	reply, err := r.posts.CreateReply(c.Request.Context(), postID, userID, pseudonym, req.Content)
	if err != nil {
		r.writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, reply)
}

func (r *Reply) list(c *gin.Context) {
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}

	limit := 20
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
			return
		}
		limit = parsed
	}

	replies, err := r.posts.ListReplies(c.Request.Context(), postID, limit)
	if err != nil {
		r.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"replies": replies})
}

func (r *Reply) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
	case errors.Is(err, domain.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrUnauthorized.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
