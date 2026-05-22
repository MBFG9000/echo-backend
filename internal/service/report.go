package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/echo-app/echo/internal/domain"
	"github.com/echo-app/echo/internal/realtime"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Report struct {
	reports           domain.ReportRepository
	posts             domain.PostRepository
	publisher         *realtime.Publisher
	redis             *redis.Client
	autoHideThreshold int
}

func NewReport(
	reports domain.ReportRepository,
	posts domain.PostRepository,
	publisher *realtime.Publisher,
	redisClient *redis.Client,
	autoHideThreshold int,
) *Report {
	return &Report{
		reports:           reports,
		posts:             posts,
		publisher:         publisher,
		redis:             redisClient,
		autoHideThreshold: autoHideThreshold,
	}
}

func (r *Report) Create(ctx context.Context, postID, reporterID uuid.UUID, reason string) (bool, error) {
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" || len(trimmed) > 500 {
		return false, domain.ErrInvalidInput
	}

	if _, err := r.posts.GetByID(ctx, postID); err != nil {
		return false, err
	}

	report := &domain.Report{
		ID:         uuid.New(),
		PostID:     postID,
		ReporterID: reporterID,
		Reason:     trimmed,
	}

	autoHidden, err := r.reports.Create(ctx, report, r.autoHideThreshold)
	if err != nil {
		return false, err
	}

	if autoHidden {
		_ = r.publisher.Publish(ctx, realtime.EventPostHidden, realtime.PostIDPayload{PostID: postID.String()})
	}

	return autoHidden, nil
}

func (r *Report) ListOpen(ctx context.Context, limit, offset int) ([]domain.Report, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	return r.reports.ListOpen(ctx, limit, offset)
}

func (r *Report) Act(ctx context.Context, adminID, reportID uuid.UUID, action domain.ModerationAction, note string) error {
	if action != domain.ModerationDismiss && action != domain.ModerationHide && action != domain.ModerationBan {
		return domain.ErrInvalidInput
	}

	report, err := r.reports.GetByID(ctx, reportID)
	if err != nil {
		return err
	}
	if report.Status != domain.ReportStatusOpen {
		return domain.ErrConflict
	}

	var authorID uuid.UUID
	if action == domain.ModerationBan {
		authorID, err = r.reports.GetPostAuthorID(ctx, report.PostID)
		if err != nil {
			return err
		}
	}

	if action == domain.ModerationHide || action == domain.ModerationBan {
		if err := r.posts.SetHidden(ctx, report.PostID, true); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.ErrNotFound
			}
			return fmt.Errorf("hide post: %w", err)
		}

		_ = r.publisher.Publish(ctx, realtime.EventPostHidden, realtime.PostIDPayload{PostID: report.PostID.String()})
	}

	if err := r.reports.Resolve(ctx, reportID, adminID, action, strings.TrimSpace(note)); err != nil {
		return err
	}

	if action == domain.ModerationBan {
		_ = r.redis.Del(ctx, sessionKey(authorID)).Err()
	}

	return nil
}

func sessionKey(userID uuid.UUID) string {
	return "session:" + userID.String()
}
