package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/echo-app/echo/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Report struct {
	db *gorm.DB
}

func NewReport(db *gorm.DB) *Report {
	return &Report{db: db}
}

func (r *Report) Create(ctx context.Context, report *domain.Report, autoHideThreshold int) (bool, error) {
	if report.ID == uuid.Nil {
		report.ID = uuid.New()
	}
	if report.CreatedAt.IsZero() {
		report.CreatedAt = time.Now()
	}
	report.Status = domain.ReportStatusOpen

	autoHidden := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Action", "ActionNote", "ReviewedBy", "ReviewedAt").Create(report).Error; err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				return domain.ErrConflict
			}
			return err
		}

		if autoHideThreshold <= 0 {
			return nil
		}

		var count int64
		if err := tx.Model(&domain.Report{}).
			Where("post_id = ?", report.PostID).
			Count(&count).Error; err != nil {
			return err
		}

		if count >= int64(autoHideThreshold) {
			if err := tx.Model(&domain.Post{}).
				Where("id = ?", report.PostID).
				Update("is_hidden", true).Error; err != nil {
				return err
			}
			autoHidden = true
		}

		return nil
	})
	if err != nil {
		return false, err
	}

	return autoHidden, nil
}

func (r *Report) ListOpen(ctx context.Context, limit, offset int) ([]domain.Report, error) {
	reports := make([]domain.Report, 0, limit)
	err := r.db.WithContext(ctx).
		Where("status = ?", domain.ReportStatusOpen).
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&reports).Error
	if err != nil {
		return nil, err
	}

	return reports, nil
}

func (r *Report) GetByID(ctx context.Context, reportID uuid.UUID) (*domain.Report, error) {
	var report domain.Report
	err := r.db.WithContext(ctx).First(&report, "id = ?", reportID).Error
	if err == nil {
		return &report, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}

	return nil, err
}

func (r *Report) GetPostAuthorID(ctx context.Context, postID uuid.UUID) (uuid.UUID, error) {
	type authorRow struct {
		AuthorID uuid.UUID
	}

	var row authorRow
	err := r.db.WithContext(ctx).
		Model(&domain.Post{}).
		Select("author_id").
		Where("id = ?", postID).
		Take(&row).Error
	if err == nil {
		return row.AuthorID, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return uuid.Nil, domain.ErrNotFound
	}

	return uuid.Nil, err
}

func (r *Report) Resolve(ctx context.Context, reportID, adminID uuid.UUID, action domain.ModerationAction, note string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&domain.Report{}).
		Where("id = ? AND status = ?", reportID, domain.ReportStatusOpen).
		Updates(map[string]any{
			"status":      domain.ReportStatusResolved,
			"action":      action,
			"action_note": note,
			"reviewed_by": adminID,
			"reviewed_at": now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrConflict
	}

	return nil
}
