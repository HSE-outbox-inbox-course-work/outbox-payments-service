package postgres

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventTypeMoneyTransferred EventType = "accounts.money.transferred"
)

// EventTime сериализуется в payload и используется потребителем для замера
// end-to-end задержки доставки.
type MoneyTransferEvent struct {
	TransferID  uuid.UUID `json:"transfer_id"`
	FromAccount uuid.UUID `json:"from_account"`
	ToAccount   uuid.UUID `json:"to_account"`
	Amount      int64     `json:"amount"`
	EventTime   time.Time `json:"event_time"`
}
