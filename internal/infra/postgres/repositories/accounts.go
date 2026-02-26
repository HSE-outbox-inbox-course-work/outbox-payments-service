package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"outbox-payment-service/internal/domain/models"

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

func (r *Accounts) BeginTx(ctx context.Context) (models.Tx, error) {
	return r.conn.Begin(ctx)
}

func (r *Accounts) CreateMoneyTransfer(ctx context.Context, tx models.Tx, in *models.TransferMoneyIn) error {
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

func (r *Accounts) MoveMoney(ctx context.Context, tx models.Tx, in *models.TransferMoneyIn) error {
	query := `
		update accounts
		set balance = case
			when id = $1 then balance - $3
			when id = $2 then balance + $3
		end
		where id in  ($1, $2);
	`

	pgTx, ok := tx.(pgx.Tx)
	if !ok {
		return errors.New("transaction is not pgx.Tx")
	}

	if _, err := pgTx.Exec(ctx, query, in.FromAccount, in.ToAccount, in.Amount); err != nil {
		return fmt.Errorf("cannot change balance: %w", err)
	}

	return nil
}

func (r *Accounts) GetByID(ctx context.Context, tx models.Tx, id models.AccountID) (*models.Account, error) {
	query := `
		select id, username, balance from accounts where accounts.id = $1 for share;
	`

	pgTx, ok := tx.(pgx.Tx)
	if !ok {
		return nil, errors.New("transaction is not pgx.Tx")
	}

	var acc models.Account
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

func (r *Accounts) createMoneyTransferredEvent(ctx context.Context, tx models.Tx, event *models.TransferMoneyIn) error {
	query := `
		insert into outbox (id, event_type, payload, status)
		values (id, event_type, payload, 'new')
	`

	pgTx, ok := tx.(pgx.Tx)
	if !ok {
		return errors.New("transaction is not pgx.Tx")
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("cannot marshal event: %w", err)
	}

	if _, err := pgTx.Exec(ctx, query, uuid.New(), models.EventTypeMoneyTransferred, payload); err != nil {
		return fmt.Errorf("cannot exec query: %w", err)
	}

	return nil
}
