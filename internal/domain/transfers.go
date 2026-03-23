package domain

import "github.com/google/uuid"

type TransferID uuid.UUID

type TransferMoneyIn struct {
	FromAccount AccountID
	ToAccount   AccountID
	Amount      int64
}
