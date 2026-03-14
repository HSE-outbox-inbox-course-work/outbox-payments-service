package outboxrely

import (
	"context"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"log"
	"outbox-payment-service/internal/infra/postgres"
	"time"
)

type EventsProvider interface {
	GetNewEvents(ctx context.Context, limit int64) ([]postgres.EventToSend, error)
	MarkSent(ctx context.Context, eventID uuid.UUID) error
}

type Worker struct {
	repo     EventsProvider
	writer   *kafka.Writer
	limit    int64
	interval time.Duration
}

func NewWorker(repo EventsProvider, writer *kafka.Writer, limit int64, interval time.Duration) *Worker {
	return &Worker{
		repo:     repo,
		writer:   writer,
		limit:    limit,
		interval: interval,
	}
}

func (w *Worker) Run(ctx context.Context) {
	t := time.NewTicker(w.interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			events, err := w.repo.GetNewEvents(ctx, w.limit) // todo в любом случае нужна транзакция чтобы два воркера одновременно могли работать
			if err != nil {
				log.Printf("error fetching events: %v", err)
				continue
			}

			for _, e := range events {
				err := w.writer.WriteMessages(ctx, kafka.Message{
					Topic: string(e.Type),
					Key:   e.ID[:],
					Value: e.Payload,
				})
				if err != nil {
					log.Printf("error writing message to kafka: %v", err)
					continue
				}

				if err := w.repo.MarkSent(ctx, e.ID); err != nil {
					log.Printf("error marking event done: %v", err)
				}
			}
		}
	}
}
