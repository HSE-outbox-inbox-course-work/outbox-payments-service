package models

import (
	"time"

	"github.com/google/uuid"
)

type TransferID uuid.UUID

type Transfer struct {
	ID          TransferID
	FromAccount uuid.UUID
	ToAccount   uuid.UUID
	Amount      int64
	CreatedAt   time.Time
}

type CreateTransferIn struct {
	FromAccount uuid.UUID
	ToAccount   uuid.UUID
	Amount      int64
}
