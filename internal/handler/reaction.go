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
	PostID string              `json:"postId" binding:"required"`
	Kind   domain.ReactionKind `json:"kind" binding:"required"`
}

func NewReaction(posts domain.PostService) *Reaction {
	return &Reaction{posts: posts}
}

func (r *Reaction) Register(rg *gin.RouterGroup) {
	rg.POST("/react", r.react)
}

// @Summary React to post
// @Tags reactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body reactRequest true "Reaction payload"
// @Success 200 {object} okResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /posts/react [post]
func (r *Reaction) react(c *gin.Context) {
	var req reactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	postID, err := uuid.Parse(req.PostID)
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
