package repository

import (
	"context"
	"time"

	"github.com/echo-app/echo/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Outbox struct {
	db *gorm.DB
}

func NewOutbox(db *gorm.DB) *Outbox {
	return &Outbox{db: db}
}

func (o *Outbox) Enqueue(ctx context.Context, eventType string, payload []byte) error {
	event := domain.OutboxEvent{
		ID:        uuid.New(),
		EventType: eventType,
		Payload:   payload,
		CreatedAt: time.Now(),
	}
	return o.db.WithContext(ctx).Create(&event).Error
}

func (o *Outbox) FetchUnprocessed(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	if limit <= 0 {
		limit = 50
	}

	events := make([]domain.OutboxEvent, 0, limit)
	err := o.db.WithContext(ctx).
		Where("processed_at IS NULL").
		Order("created_at ASC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

func (o *Outbox) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return o.db.WithContext(ctx).
		Model(&domain.OutboxEvent{}).
		Where("id = ?", id).
		Update("processed_at", now).Error
}

func (o *Outbox) EnqueueTx(tx *gorm.DB, eventType string, payload []byte) error {
	event := domain.OutboxEvent{
		ID:        uuid.New(),
		EventType: eventType,
		Payload:   payload,
		CreatedAt: time.Now(),
	}
	return tx.Create(&event).Error
}
