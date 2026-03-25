package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/echo-app/echo/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Admin struct {
	reports domain.ReportService
}

type moderateRequest struct {
	Action domain.ModerationAction `json:"action" binding:"required"`
	Note   string                  `json:"note"`
}

type reportView struct {
	ID         uuid.UUID               `json:"id"`
	PostID     uuid.UUID               `json:"postId"`
	Reason     string                  `json:"reason"`
	Status     domain.ReportStatus     `json:"status"`
	Action     domain.ModerationAction `json:"action"`
	ActionNote string                  `json:"actionNote"`
	ReviewedBy *uuid.UUID              `json:"reviewedBy"`
	ReviewedAt *string                 `json:"reviewedAt"`
	CreatedAt  string                  `json:"createdAt"`
}

func NewAdmin(reports domain.ReportService) *Admin {
	return &Admin{reports: reports}
}

func (a *Admin) Register(rg *gin.RouterGroup) {
	rg.GET("/reports", a.listReports)
	rg.POST("/reports/:id/action", a.action)
}

func (a *Admin) listReports(c *gin.Context) {
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
			return
		}
		limit = parsed
	}

	offset := 0
	if raw := c.Query("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
			return
		}
		offset = parsed
	}

	reports, err := a.reports.ListOpen(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	items := make([]reportView, 0, len(reports))
	for _, report := range reports {
		items = append(items, toReportView(report))
	}

	c.JSON(http.StatusOK, gin.H{"reports": items})
}

func (a *Admin) action(c *gin.Context) {
	reportID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}

	adminIDValue, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrUnauthorized.Error()})
		return
	}

	adminID, ok := adminIDValue.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrUnauthorized.Error()})
		return
	}

	var req moderateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}

	if err := a.reports.Act(c.Request.Context(), adminID, reportID, req.Action, req.Note); err != nil {
		a.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *Admin) writeError(c *gin.Context, err error) {
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

func toReportView(report domain.Report) reportView {
	createdAt := report.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
	var reviewedAt *string
	if report.ReviewedAt != nil {
		value := report.ReviewedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
		reviewedAt = &value
	}

	return reportView{
		ID:         report.ID,
		PostID:     report.PostID,
		Reason:     report.Reason,
		Status:     report.Status,
		Action:     report.Action,
		ActionNote: report.ActionNote,
		ReviewedBy: report.ReviewedBy,
		ReviewedAt: reviewedAt,
		CreatedAt:  createdAt,
	}
}
