package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"outbox-payment-service/internal/domain/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Accounts struct {
	conn *pgxpool.Pool
}

func NewAccounts(conn *pgxpool.Pool) *Accounts {
	return &Accounts{conn: conn}
}

func (r *Accounts) BeginTx(ctx context.Context) (domain.Tx, error) {
	return r.conn.Begin(ctx)
}

func (r *Accounts) ChangeBalance(ctx context.Context, tx domain.Tx, in *domain.ChangeBalanceIn) error {
	query := `
		update accounts set balance = balance + $1 where id = $2;
	`

	pgTx, ok := tx.(pgx.Tx)
	if !ok {
		return errors.New("transaction is not r pgx.Tx")
	}

	if _, err := pgTx.Exec(ctx, query, in.Amount, in.AccountID); err != nil {
		return fmt.Errorf("cannot change balance: %w", err)
	}

	return nil
}

func (r *Accounts) CreateMoneyTransferredEvent(ctx context.Context, tx domain.Tx, event *domain.TransferMoneyIn) error {
	query := `
		insert into outbox (id, event_type, payload, status)
		values (id, event_type, payload, 'new')
	`

	pgTx, ok := tx.(pgx.Tx)
	if !ok {
		return errors.New("transaction is not r pgx.Tx")
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("cannot marshal event: %w", err)
	}

	if _, err := pgTx.Exec(ctx, query, uuid.New(), domain.EventTypeMoneyTransferred, payload); err != nil {
		return fmt.Errorf("cannot exec query: %w", err)
	}

	return nil
}

func (r *Accounts) GetByID(ctx context.Context, tx domain.Tx, id domain.AccountID) (*domain.Account, error) {
	query := `
		select id, username, balance from accounts where accounts.id = $1 for share;
	`

	pgTx, ok := tx.(pgx.Tx)
	if !ok {
		return nil, errors.New("transaction is not r pgx.Tx")
	}

	var acc domain.Account
	err := pgTx.QueryRow(ctx, query, id).Scan(
		&acc.ID,
		&acc.Username,
		&acc.Balance,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot scan account: %w", err)
	}

	return &acc, nil
}
