package postgres

import (
	"github.com/google/uuid"
)

type EventType string

const (
	EventTypeMoneyTransferred EventType = "accounts.money.transferred"
)

type MoneyTransferEvent struct {
	TransferID  uuid.UUID `json:"transfer_id"`
	FromAccount uuid.UUID `json:"from_account"`
	ToAccount   uuid.UUID `json:"to_account"`
	Amount      int64     `json:"amount"`
}
