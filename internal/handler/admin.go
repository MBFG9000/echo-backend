package handler

import (
	"net/http"

	"github.com/echo-app/echo/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Admin struct {
	reports domain.ReportService
}

type moderateRequest struct {
	ReportID string                  `json:"reportId" binding:"required"`
	Action   domain.ModerationAction `json:"action" binding:"required"`
	Note     string                  `json:"note"`
}

type listReportsRequest struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type listReportsResponse struct {
	Reports []reportView `json:"reports"`
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
	rg.POST("/reports/list", a.listReports)
	rg.POST("/reports/action", a.action)
}

// @Summary List open reports
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body listReportsRequest true "List payload"
// @Success 200 {object} listReportsResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /admin/reports/list [post]
func (a *Admin) listReports(c *gin.Context) {
	var req listReportsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	limit := 20
	if req.Limit > 0 {
		limit = req.Limit
	}

	offset := 0
	if req.Offset > 0 {
		offset = req.Offset
	}

	reports, err := a.reports.ListOpen(c.Request.Context(), limit, offset)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	items := make([]reportView, 0, len(reports))
	for _, report := range reports {
		items = append(items, toReportView(report))
	}

	c.JSON(http.StatusOK, listReportsResponse{Reports: items})
}

// @Summary Resolve report
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body moderateRequest true "Action payload"
// @Success 200 {object} okResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /admin/reports/action [post]
func (a *Admin) action(c *gin.Context) {
	adminIDValue, ok := c.Get("userID")
	if !ok {
		writeDomainError(c, domain.ErrUnauthorized)
		return
	}

	adminID, ok := adminIDValue.(uuid.UUID)
	if !ok {
		writeDomainError(c, domain.ErrUnauthorized)
		return
	}

	var req moderateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	reportID, err := uuid.Parse(req.ReportID)
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	if err := a.reports.Act(c.Request.Context(), adminID, reportID, req.Action, req.Note); err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, okResponse{OK: true})
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
