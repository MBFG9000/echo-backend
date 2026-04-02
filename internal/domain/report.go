package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ReportStatus string

type ModerationAction string

const (
	ReportStatusOpen     ReportStatus = "open"
	ReportStatusResolved ReportStatus = "resolved"

	ModerationDismiss ModerationAction = "dismiss"
	ModerationHide    ModerationAction = "hide"
	ModerationBan     ModerationAction = "ban"
)

type Report struct {
	ID         uuid.UUID        `json:"id" gorm:"type:uuid;primaryKey"`
	PostID     uuid.UUID        `json:"postId" gorm:"type:uuid;index;not null"`
	ReporterID uuid.UUID        `json:"reporterId" gorm:"type:uuid;index;not null"`
	Reason     string           `json:"reason" gorm:"type:text;not null"`
	Status     ReportStatus     `json:"status" gorm:"type:text;not null;default:open"`
	Action     ModerationAction `json:"action" gorm:"type:text"`
	ActionNote string           `json:"actionNote" gorm:"type:text;not null;default:''"`
	ReviewedBy *uuid.UUID       `json:"reviewedBy" gorm:"type:uuid"`
	ReviewedAt *time.Time       `json:"reviewedAt"`
	CreatedAt  time.Time        `json:"createdAt"`
}

type ReportRepository interface {
	Create(ctx context.Context, report *Report, autoHideThreshold int) (bool, error)
	ListOpen(ctx context.Context, limit, offset int) ([]Report, error)
	GetByID(ctx context.Context, reportID uuid.UUID) (*Report, error)
	GetPostAuthorID(ctx context.Context, postID uuid.UUID) (uuid.UUID, error)
	Resolve(ctx context.Context, reportID, adminID uuid.UUID, action ModerationAction, note string) error
}
