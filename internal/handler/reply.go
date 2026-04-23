package handler

import (
	"net/http"

	"github.com/echo-app/echo/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Reply struct {
	posts domain.PostService
}

type createReplyRequest struct {
	PostID        string `json:"postId" binding:"required"`
	ParentReplyID string `json:"parentReplyId"`
	Content       string `json:"content" binding:"required,max=280"`
}

type listRepliesRequest struct {
	PostID string `json:"postId" binding:"required"`
	Limit  int    `json:"limit"`
}

type updateReplyRequest struct {
	ReplyID string `json:"replyId" binding:"required"`
	Content string `json:"content" binding:"required,max=280"`
}

type deleteReplyRequest struct {
	ReplyID string `json:"replyId" binding:"required"`
}

type reactReplyRequest struct {
	ReplyID string              `json:"replyId" binding:"required"`
	Kind    domain.ReactionKind `json:"kind" binding:"required"`
}

func NewReply(posts domain.PostService) *Reply {
	return &Reply{posts: posts}
}

func (r *Reply) RegisterPublic(rg *gin.RouterGroup) {
	rg.POST("/replies/list", r.list)
}

func (r *Reply) RegisterPrivate(rg *gin.RouterGroup) {
	rg.POST("/replies/create", r.create)
	rg.POST("/replies/update", r.update)
	rg.POST("/replies/delete", r.delete)
	rg.POST("/replies/react", r.react)
}

func (r *Reply) create(c *gin.Context) {
	var req createReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	postID, err := uuid.Parse(req.PostID)
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	var parentReplyID *uuid.UUID
	if req.ParentReplyID != "" {
		parsed, parseErr := uuid.Parse(req.ParentReplyID)
		if parseErr != nil {
			writeDomainError(c, domain.ErrInvalidInput)
			return
		}
		parentReplyID = &parsed
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

	reply, err := r.posts.CreateReply(c.Request.Context(), postID, parentReplyID, userID, pseudonym, req.Content)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, reply)
}

func (r *Reply) list(c *gin.Context) {
	var req listRepliesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	postID, err := uuid.Parse(req.PostID)
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	limit := 20
	if req.Limit > 0 {
		limit = req.Limit
	}

	replies, err := r.posts.ListReplies(c.Request.Context(), postID, limit)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"replies": replies})
}

func (r *Reply) update(c *gin.Context) {
	var req updateReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	replyID, err := uuid.Parse(req.ReplyID)
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

	reply, err := r.posts.UpdateReply(c.Request.Context(), replyID, userID, req.Content)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, reply)
}

func (r *Reply) delete(c *gin.Context) {
	var req deleteReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	replyID, err := uuid.Parse(req.ReplyID)
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

	if err := r.posts.DeleteReply(c.Request.Context(), replyID, userID); err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (r *Reply) react(c *gin.Context) {
	var req reactReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	replyID, err := uuid.Parse(req.ReplyID)
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

	if err := r.posts.ReactReply(c.Request.Context(), replyID, userID, req.Kind); err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
