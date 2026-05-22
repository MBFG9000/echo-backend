package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/echo-app/echo/internal/domain"
	"github.com/echo-app/echo/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Reply struct {
	posts domain.PostService
	auth  *middleware.Auth
}

type createReplyRequest struct {
	PostID        string `json:"postId" binding:"required"`
	ParentReplyID string `json:"parentReplyId"`
	Content       string `json:"content" binding:"required,max=280"`
}

type createReplyBody struct {
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

type listRepliesResponse struct {
	Replies []domain.Reply `json:"replies"`
}

func NewReply(posts domain.PostService, auth *middleware.Auth) *Reply {
	return &Reply{posts: posts, auth: auth}
}

func (r *Reply) RegisterPublic(rg *gin.RouterGroup) {
	rg.GET("/:id/replies", r.listFromParam)
	rg.POST("/replies/list", r.list)
}

func (r *Reply) RegisterPrivate(rg *gin.RouterGroup) {
	rg.POST("/:id/replies", r.createFromParam)
	rg.POST("/replies/create", r.create)
	rg.POST("/replies/update", r.update)
	rg.POST("/replies/delete", r.delete)
	rg.POST("/replies/:replyId/react", r.reactFromParam)
	rg.DELETE("/replies/:replyId/react", r.unreactFromParam)
	rg.POST("/replies/react", r.react)
}

// @Summary Create reply
// @Tags replies
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body createReplyRequest true "Reply payload"
// @Success 201 {object} domain.Reply
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /posts/replies/create [post]
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

	parentReplyID, err := parseOptionalReplyID(req.ParentReplyID)
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	r.writeCreatedReply(c, postID, parentReplyID, req.Content)
}

func (r *Reply) createFromParam(c *gin.Context) {
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	var req createReplyBody
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	parentReplyID, err := parseOptionalReplyID(req.ParentReplyID)
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	r.writeCreatedReply(c, postID, parentReplyID, req.Content)
}

func (r *Reply) writeCreatedReply(c *gin.Context, postID uuid.UUID, parentReplyID *uuid.UUID, content string) {
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

	reply, err := r.posts.CreateReply(c.Request.Context(), postID, parentReplyID, userID, pseudonym, content)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, reply)
}

func (r *Reply) listFromParam(c *gin.Context) {
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	limit := 20
	if raw := c.Query("limit"); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil {
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

	if r.auth != nil {
		if userID, ok := r.auth.TryUserID(c); ok {
			r.posts.MarkViewerReactionsOnReplies(c.Request.Context(), userID, replies)
		}
	}

	c.JSON(http.StatusOK, listRepliesResponse{Replies: replies})
}

// @Summary List replies for post
// @Tags replies
// @Accept json
// @Produce json
// @Param request body listRepliesRequest true "List payload"
// @Success 200 {object} listRepliesResponse
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /posts/replies/list [post]
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

	if r.auth != nil {
		if userID, ok := r.auth.TryUserID(c); ok {
			r.posts.MarkViewerReactionsOnReplies(c.Request.Context(), userID, replies)
		}
	}

	c.JSON(http.StatusOK, listRepliesResponse{Replies: replies})
}

// @Summary Update reply
// @Tags replies
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body updateReplyRequest true "Update payload"
// @Success 200 {object} domain.Reply
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /posts/replies/update [post]
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

// @Summary Delete reply
// @Tags replies
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body deleteReplyRequest true "Delete payload"
// @Success 200 {object} okResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /posts/replies/delete [post]
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

// @Summary React to reply
// @Tags replies
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body reactReplyRequest true "Reaction payload"
// @Success 200 {object} okResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /posts/replies/react [post]
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

	r.reactReply(c, replyID, req.Kind)
}

func (r *Reply) reactFromParam(c *gin.Context) {
	replyID, err := uuid.Parse(c.Param("replyId"))
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	var body struct {
		Kind domain.ReactionKind `json:"kind" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		writeValidationError(c, err)
		return
	}

	r.reactReply(c, replyID, body.Kind)
}

func (r *Reply) unreactFromParam(c *gin.Context) {
	replyID, err := uuid.Parse(c.Param("replyId"))
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	userID, ok := userIDFromContext(c)
	if !ok {
		return
	}

	if err := r.posts.UnreactReply(c.Request.Context(), replyID, userID); err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (r *Reply) reactReply(c *gin.Context, replyID uuid.UUID, kind domain.ReactionKind) {
	userID, ok := userIDFromContext(c)
	if !ok {
		return
	}

	if err := r.posts.ReactReply(c.Request.Context(), replyID, userID, kind); err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func parseOptionalReplyID(raw string) (*uuid.UUID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}
