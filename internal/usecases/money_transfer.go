package usecases

import (
	"context"
	"errors"
	"fmt"
	"outbox-payment-service/internal/domain"
)

type accountsRepository interface {
	BeginTx(context.Context) (domain.Tx, error)
	CreateMoneyTransfer(context.Context, domain.Tx, *domain.TransferMoneyIn) error
	GetByID(context.Context, domain.Tx, domain.AccountID) (*domain.Account, error)
	UpdateAccountBalance(context.Context, domain.Tx, domain.AccountID, int64) error
}

type MoneyTransfer struct {
	accountsRepository accountsRepository
}

func NewMoneyTransfer(accountsRepository accountsRepository) *MoneyTransfer {
	return &MoneyTransfer{
		accountsRepository: accountsRepository,
	}
}

func (u *MoneyTransfer) TransferMoney(ctx context.Context, in *domain.TransferMoneyIn) (err error) {
	if in.Amount <= 0 {
		return domain.ErrInvalidMoneyTransferAmount
	}

	panic("13")

	tx, err := u.accountsRepository.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("cannot begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, tx.Rollback(ctx))
		} else {
			err = tx.Commit(ctx)
		}
	}()

	from, err := u.accountsRepository.GetByID(ctx, tx, in.FromAccount)
	if err != nil {
		return fmt.Errorf("cannot get from account: %w", err)
	}

	to, err := u.accountsRepository.GetByID(ctx, tx, in.ToAccount)
	if err != nil {
		return fmt.Errorf("cannot get to account: %w", err)
	}

	if from.Balance < in.Amount {
		return domain.ErrInsufficientFunds
	}

	if err = u.accountsRepository.UpdateAccountBalance(ctx, tx, from.ID, -in.Amount); err != nil {
		return fmt.Errorf("cannot move money: %w", err)
	}

	if err = u.accountsRepository.UpdateAccountBalance(ctx, tx, to.ID, in.Amount); err != nil {
		return fmt.Errorf("cannot move money: %w", err)
	}

	if err := u.accountsRepository.CreateMoneyTransfer(ctx, tx, in); err != nil {
		return fmt.Errorf("cannot create transfer event: %w", err)
	}

	return nil
}
