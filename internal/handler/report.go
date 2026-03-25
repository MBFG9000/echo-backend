package handler

import (
	"errors"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}

	var req createReportRequest
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

	autoHidden, err := r.reports.Create(c.Request.Context(), postID, userID, req.Reason)
	if err != nil {
		r.writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"ok": true, "auto_hidden": autoHidden})
}

func (r *Report) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
	case errors.Is(err, domain.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"error": domain.ErrConflict.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
