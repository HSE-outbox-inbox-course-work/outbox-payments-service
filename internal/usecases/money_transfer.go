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
	MoveMoney(context.Context, domain.Tx, *domain.TransferMoneyIn) error
}

type MoneyTransfer struct {
	accountsRepository accountsRepository
}

func NewMoneyTransfer(accountsRepository accountsRepository) *MoneyTransfer {
	return &MoneyTransfer{
		accountsRepository: accountsRepository,
	}
}

// проблемы
// нет проверки что accountTo существует
// если его не существует то деньги спишуться но никуда не зачисляться (нужно исправить запрос в moveMoney)
// если делать GetById сначала from потом to может быть дедлок
// Не проверяется результат UPDATE Нужно смотреть RowsAffected или использовать CTE с RETURNING, чтобы понимать, списались ли деньги и зачислились ли.

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

	from, err := u.accountsRepository.GetByID(ctx, tx, in.FromAccount)
	if err != nil {
		return fmt.Errorf("cannot get from account: %w", err)
	}

	if in.Amount <= 0 {
		return domain.ErrInvalidMoneyTransferAmount
	}

	if from.Balance < in.Amount {
		return domain.ErrInsufficientFunds
	}

	if err = u.accountsRepository.MoveMoney(ctx, tx, in); err != nil {
		return fmt.Errorf("cannot move money: %w", err)
	}

	if err := u.accountsRepository.CreateMoneyTransfer(ctx, tx, in); err != nil {
		return fmt.Errorf("cannot create transfer event: %w", err)
	}

	return nil
}
