package handler

import (
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
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	var req createReplyRequest
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

	reply, err := r.posts.CreateReply(c.Request.Context(), postID, userID, pseudonym, req.Content)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, reply)
}

func (r *Reply) list(c *gin.Context) {
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	limit := 20
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeDomainError(c, domain.ErrInvalidInput)
			return
		}
		limit = parsed
	}

	replies, err := r.posts.ListReplies(c.Request.Context(), postID, limit)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"replies": replies})
}
