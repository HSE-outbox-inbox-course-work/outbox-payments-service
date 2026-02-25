package accounts

import (
	"context"
	"errors"
	"fmt"
	"outbox-payment-service/internal/domain/domain"
)

type accountsRepository interface {
	BeginTx(context.Context) (domain.Tx, error)
	ChangeBalance(context.Context, domain.Tx, *domain.ChangeBalanceIn) error
	GetByID(context.Context, domain.Tx, domain.AccountID) (*domain.Account, error)
	CreateMoneyTransferredEvent(context.Context, domain.Tx, *domain.TransferMoneyIn) error
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

	account, err := u.accountsRepository.GetByID(ctx, tx, in.FromAccount)
	if err != nil {
		return fmt.Errorf("cannot get account: %w", err)
	}

	if account.Balance < in.Amount {
		return domain.ErrInsufficientFunds
	}

	err = u.accountsRepository.ChangeBalance(ctx, tx, &domain.ChangeBalanceIn{
		AccountID: in.FromAccount,
		Amount:    -in.Amount,
	})
	if err != nil {
		return fmt.Errorf("cannot decrease account balance: %w", err)
	}

	err = u.accountsRepository.ChangeBalance(ctx, tx, &domain.ChangeBalanceIn{
		AccountID: in.ToAccount,
		Amount:    in.Amount,
	})
	if err != nil {
		return fmt.Errorf("cannot increase account balance: %w", err)
	}

	if err := u.accountsRepository.CreateMoneyTransferredEvent(ctx, tx, in); err != nil {
		return fmt.Errorf("cannot create transfer event: %w", err)
	}

	return nil
}
