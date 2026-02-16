package transfers

import (
	"context"
	"errors"
	"fmt"
	"outbox-payment-service/internal/domain/models"
)

type transfersRepository interface {
	Begin(context.Context) (models.Tx, error)
	CreateTransfer(context.Context, models.Tx, *models.CreateTransferIn) (models.TransferID, error)
	CreateTransferNotificationTask(context.Context, models.Tx, models.TransferID) error
}

type Creator struct {
	transfersRepository transfersRepository
}

func NewCreator(transfersRepository transfersRepository) *Creator {
	return &Creator{
		transfersRepository: transfersRepository,
	}
}

func (u *Creator) Create(ctx context.Context, in *models.CreateTransferIn) (err error) {
	tx, err := u.transfersRepository.Begin(ctx)
	if err != nil {
		return fmt.Errorf("cannot begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				err = errors.Join(err, rollbackErr)
			}
		} else {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				err = commitErr
			}
		}
	}()

	transferID, err := u.transfersRepository.CreateTransfer(ctx, tx, in)
	if err != nil {
		return fmt.Errorf("cannot create transfer: %w", err)
	}

	if err = u.transfersRepository.CreateTransferNotificationTask(ctx, tx, transferID); err != nil {
		return fmt.Errorf("cannot create transfer notification task: %w", err)
	}

	return nil
}
