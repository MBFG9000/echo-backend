package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/echo-app/echo/internal/domain"
)

type Broadcaster interface {
	Broadcast(payload []byte)
}

type Outbox struct {
	outbox      domain.OutboxRepository
	broadcaster Broadcaster
	logger      *slog.Logger
	interval    time.Duration
}

func NewOutbox(outbox domain.OutboxRepository, broadcaster Broadcaster, logger *slog.Logger) *Outbox {
	if logger == nil {
		logger = slog.Default()
	}
	return &Outbox{
		outbox:      outbox,
		broadcaster: broadcaster,
		interval:    500 * time.Millisecond,
	}
}

func (w *Outbox) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.flush(ctx)
		}
	}
}

func (w *Outbox) flush(ctx context.Context) {
	events, err := w.outbox.FetchUnprocessed(ctx, 50)
	if err != nil {
		w.logger.Error("outbox_fetch_failed", slog.String("error", err.Error()))
		return
	}

	for _, event := range events {
		switch event.EventType {
		case domain.OutboxEventPostCreated:
			if w.broadcaster != nil {
				w.broadcaster.Broadcast(event.Payload)
			}
		default:
			w.logger.Warn("outbox_unknown_event", slog.String("type", event.EventType))
		}

		if err := w.outbox.MarkProcessed(ctx, event.ID); err != nil {
			w.logger.Error("outbox_mark_processed_failed", slog.String("error", err.Error()))
			return
		}
	}
}
