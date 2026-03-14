package postgres

import (
	"encoding/json"
	"github.com/google/uuid"
)

type EventType string

const (
	EventTypeMoneyTransferred EventType = "accounts.money.transferred" //todo вообще так себе разделение точками на доменное что то
)

type MoneyTransferEvent struct {
	FromAccount uuid.UUID `json:"from_account"`
	ToAccount   uuid.UUID `json:"to_account"`
	Amount      int64     `json:"amount"`
}

type EventToSend struct {
	ID      uuid.UUID
	Payload json.RawMessage
	Type    EventType
}
