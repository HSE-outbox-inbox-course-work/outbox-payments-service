package models

import (
	"encoding/json"

	"github.com/google/uuid"
)

type EventID uuid.UUID

type EventType string

const (
	EventTypeMoneyTransferred EventType = "money_transfer"
)

type Event struct {
	ID      EventID
	Type    EventType
	Payload json.RawMessage
}
