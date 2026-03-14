package repositories

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"outbox-payment-service/internal/infra/postgres"
)

type Outbox struct {
	conn *pgxpool.Pool
}

func NewOutbox(conn *pgxpool.Pool) *Outbox {
	return &Outbox{conn: conn}
}

func (r *Outbox) GetNewEvents(ctx context.Context, limit int64) ([]postgres.EventToSend, error) {
	query := `
		select id, event_type, payload from outbox
	    where status = 'new'
	    order by created_at
		limit $1
	`

	rows, err := r.conn.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("cannot get rows%w", err)
	}

	return pgx.CollectRows(rows, r.collectEvent)
}

func (r *Outbox) MarkSent(ctx context.Context, id uuid.UUID) error {
	query := `
        update outbox
        set status = 'done'
        where id = $1
    `

	if _, err := r.conn.Exec(ctx, query, id); err != nil {
		return fmt.Errorf("cannot mark event done: %w", err)
	}

	return nil
}

func (r *Outbox) collectEvent(row pgx.CollectableRow) (postgres.EventToSend, error) {
	var event postgres.EventToSend
	err := row.Scan(
		&event.ID,
		&event.Type,
		&event.Payload,
	)
	return event, err
}
