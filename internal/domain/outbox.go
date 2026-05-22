package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const OutboxEventPostCreated = "post.created"

type OutboxEvent struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	EventType   string     `json:"eventType" gorm:"not null"`
	Payload     []byte     `json:"-" gorm:"type:bytea;not null"`
	CreatedAt   time.Time  `json:"createdAt"`
	ProcessedAt *time.Time `json:"processedAt,omitempty"`
}

type OutboxRepository interface {
	Enqueue(ctx context.Context, eventType string, payload []byte) error
	FetchUnprocessed(ctx context.Context, limit int) ([]OutboxEvent, error)
	MarkProcessed(ctx context.Context, id uuid.UUID) error
}
