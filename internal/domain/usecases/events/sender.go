package events

import (
	"context"
	"encoding/json"
	"outbox-payment-service/internal/domain/domain"
	"time"
)

type eventHandler interface {
	Handle(ctx context.Context, payload json.RawMessage) error
}

type eventsRepository interface {
	GetPending(ctx context.Context, limit int) ([]domain.Event, error)
	MarkSent(ctx context.Context, id domain.EventID) error
	MarkFailed(ctx context.Context, id domain.EventID, err string, nextRetryAt time.Time) error
}

type Sender struct {
	handlers         map[domain.EventType]eventHandler
	eventsRepository eventsRepository
}

func NewSender(handlers map[domain.EventType]eventHandler) *Sender {
	return &Sender{
		handlers: handlers,
	}
}

func (s *Sender) Send(ctx context.Context) error {
	events, err := s.eventsRepository.GetPending(ctx, 100)
	if err != nil {
		return err
	}

	for _, event := range events {
		handler, ok := s.handlers[event.Type]
		if !ok {
			_ = s.eventsRepository.MarkFailed(ctx, event.ID, "no handler", time.Now())
			continue
		}

		if err := handler.Handle(ctx, event.Payload); err != nil {
			_ = s.eventsRepository.MarkFailed(ctx, event.ID, err.Error(), time.Now().Add(time.Minute))
			continue
		}

		if err := s.eventsRepository.MarkSent(ctx, event.ID); err != nil {
			return err
		}
	}

	return nil
}
