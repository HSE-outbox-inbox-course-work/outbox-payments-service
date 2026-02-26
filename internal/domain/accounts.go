package domain

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrInsufficientFunds          = errors.New("insufficient funds")
	ErrInvalidMoneyTransferAmount = errors.New("invalid money transfer amount")
	ErrAccountNotFound            = errors.New("account not found")
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
