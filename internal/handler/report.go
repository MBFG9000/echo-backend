package handler

import (
	"net/http"

	"github.com/echo-app/echo/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Report struct {
	reports domain.ReportService
}

type createReportRequest struct {
	PostID string `json:"postId" binding:"required"`
	Reason string `json:"reason" binding:"required,max=500"`
}

type createReportResponse struct {
	OK         bool `json:"ok"`
	AutoHidden bool `json:"autoHidden"`
}

func NewReport(reports domain.ReportService) *Report {
	return &Report{reports: reports}
}

func (r *Report) RegisterPrivate(rg *gin.RouterGroup) {
	rg.POST("/:id/report", r.createFromParam)
	rg.POST("/report", r.create)
}

type createReportByIDRequest struct {
	Reason string `json:"reason" binding:"required,max=500"`
}

// @Summary Report post
// @Tags reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body createReportRequest true "Report payload"
// @Success 201 {object} createReportResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /posts/{id}/report [post]
func (r *Report) createFromParam(c *gin.Context) {
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	var req createReportByIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	r.createReport(c, postID, req.Reason)
}

// @Summary Report post (legacy body)
// @Tags reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body createReportRequest true "Report payload"
// @Success 201 {object} createReportResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /posts/report [post]
func (r *Report) create(c *gin.Context) {
	var req createReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	postID, err := uuid.Parse(req.PostID)
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	r.createReport(c, postID, req.Reason)
}

func (r *Report) createReport(c *gin.Context, postID uuid.UUID, reason string) {
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

	autoHidden, err := r.reports.Create(c.Request.Context(), postID, userID, reason)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, createReportResponse{OK: true, AutoHidden: autoHidden})
}
