package realtime

import (
	"context"

	"github.com/echo-app/echo/internal/domain"
)

type Publisher struct {
	outbox domain.OutboxRepository
}

func NewPublisher(outbox domain.OutboxRepository) *Publisher {
	return &Publisher{outbox: outbox}
}

func (p *Publisher) Publish(ctx context.Context, eventType string, payload any) error {
	if p == nil || p.outbox == nil {
		return nil
	}

	return Publish(p.outbox, ctx, eventType, payload)
}
