package handler

import (
	"net/http"

	"github.com/echo-app/echo/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Reaction struct {
	posts domain.PostService
}

type reactRequest struct {
	Kind domain.ReactionKind `json:"kind" binding:"required"`
}

func NewReaction(posts domain.PostService) *Reaction {
	return &Reaction{posts: posts}
}

func (r *Reaction) Register(rg *gin.RouterGroup) {
	rg.POST("/:id/react", r.react)
}

func (r *Reaction) react(c *gin.Context) {
	var req reactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
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

	if err := r.posts.React(c.Request.Context(), postID, userID, req.Kind); err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
