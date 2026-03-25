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
	Reason string `json:"reason" binding:"required,max=500"`
}

func NewReport(reports domain.ReportService) *Report {
	return &Report{reports: reports}
}

func (r *Report) RegisterPrivate(rg *gin.RouterGroup) {
	rg.POST("/:id/report", r.create)
}

func (r *Report) create(c *gin.Context) {
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	var req createReportRequest
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

	autoHidden, err := r.reports.Create(c.Request.Context(), postID, userID, req.Reason)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"ok": true, "auto_hidden": autoHidden})
}
