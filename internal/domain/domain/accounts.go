package domain

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
)

type AccountID uuid.UUID

type Account struct {
	ID       AccountID
	Username string
	Balance  int64
}

type TransferMoneyIn struct {
	FromAccount AccountID
	ToAccount   AccountID
	Amount      int64
}

type ChangeBalanceIn struct {
	AccountID AccountID
	Amount    int64
}
