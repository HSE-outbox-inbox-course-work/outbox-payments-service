package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"outbox-payment-service/internal/domain"
	"outbox-payment-service/internal/infra/postgres"

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

func (r *Accounts) CreateMoneyTransfer(ctx context.Context, tx domain.Tx, in *domain.TransferMoneyIn) error {
	query := `
		insert into transfers(id, from_account_id, to_account_id, amount)
		values ($1, $2, $3, $4)
	`

	pgTx, ok := tx.(pgx.Tx)
	if !ok {
		return errors.New("transaction is not pgx.Tx")
	}

	if _, err := pgTx.Exec(ctx, query, in.FromAccount, in.ToAccount, in.Amount); err != nil {
		return fmt.Errorf("cannot create money transfer: %w", err)
	}

	if err := r.createMoneyTransferredEvent(ctx, tx, in); err != nil {
		return fmt.Errorf("cannot create money transfer event: %w", err)
	}

	return nil
}

func (r *Accounts) UpdateAccountBalance(ctx context.Context, tx domain.Tx, id domain.AccountID, amount int64) error {
	query := `
		update accounts
		set balance = balance + $1
		where id = $2
	`

	pgTx, ok := tx.(pgx.Tx)
	if !ok {
		return errors.New("transaction is not pgx.Tx")
	}

	res, err := pgTx.Exec(ctx, query, amount, id)
	if err != nil {
		return fmt.Errorf("cannot change balance: %w", err)
	}
	if res.RowsAffected() != 1 {
		return errors.New("cannot change balance rows affect not 1")
	}

	return nil
}

func (r *Accounts) GetByID(ctx context.Context, tx domain.Tx, id domain.AccountID) (*domain.Account, error) {
	query := `
		select id, username, balance from accounts where accounts.id = $1 for update;
	`

	pgTx, ok := tx.(pgx.Tx)
	if !ok {
		return nil, errors.New("transaction is not pgx.Tx")
	}

	var acc domain.Account
	err := pgTx.QueryRow(ctx, query, id).Scan(
		&acc.ID,
		&acc.Username,
		&acc.Balance,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrAccountNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("cannot scan account: %w", err)
	}

	return &acc, nil
}

func (r *Accounts) createMoneyTransferredEvent(ctx context.Context, tx domain.Tx, event *domain.TransferMoneyIn) error {
	query := `
		insert into outbox (id, event_type, payload, status)
		values ($1, $2, $3, 'new')
	`

	pgTx, ok := tx.(pgx.Tx)
	if !ok {
		return errors.New("transaction is not pgx.Tx")
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("cannot marshal event: %w", err)
	}

	if _, err := pgTx.Exec(ctx, query, uuid.New(), postgres.EventTypeMoneyTransferred, payload); err != nil {
		return fmt.Errorf("cannot exec query: %w", err)
	}

	return nil
}
