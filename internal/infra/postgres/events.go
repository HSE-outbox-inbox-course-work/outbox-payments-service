package postgres

import (
	"github.com/google/uuid"
)

type EventType string

const (
	EventTypeMoneyTransferred EventType = "money_transfer"
)

type MoneyTransferEvent struct {
	FromAccount uuid.UUID `json:"from_account"`
	ToAccount   uuid.UUID `json:"to_account"`
	Amount      int64     `json:"amount"`
}
