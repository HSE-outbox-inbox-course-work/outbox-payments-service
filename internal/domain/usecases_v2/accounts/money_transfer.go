package accounts

import (
	"context"
	"errors"
	"fmt"
	"outbox-payment-service/internal/domain/models"
)

type accountsRepository interface {
	BeginTx(context.Context) (models.Tx, error)
	TransferMoney(context.Context, models.Tx, *models.ChangeBalanceIn) error
	GetByID(context.Context, models.Tx, models.AccountID) (*models.Account, error)
	CreateMoneyTransferredEvent(context.Context, models.Tx, *models.TransferMoneyIn) error
}

type MoneyTransfer struct {
	accountsRepository accountsRepository
}

func NewMoneyTransfer(accountsRepository accountsRepository) *MoneyTransfer {
	return &MoneyTransfer{
		accountsRepository: accountsRepository,
	}
}

func (u *MoneyTransfer) TransferMoney(ctx context.Context, in *models.TransferMoneyIn) (err error) {
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
		return models.ErrInsufficientFunds
	}

	err = u.accountsRepository.TransferMoney(ctx, tx, &models.ChangeBalanceIn{
		AccountID: in.FromAccount,
		Amount:    -in.Amount,
	})
	if err != nil {
		return fmt.Errorf("cannot decrease account balance: %w", err)
	}

	err = u.accountsRepository.TransferMoney(ctx, tx, &models.ChangeBalanceIn{
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
