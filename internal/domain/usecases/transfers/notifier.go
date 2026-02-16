package transfers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"outbox-payment-service/internal/domain/models"
	"sync"
)

type transferNotificationsOutbox interface {
	Begin(context.Context) (models.Tx, error)
	ClaimUnsentTransferNotificationTasks(context.Context, models.Tx, int64) ([]models.Transfer, error)
	MarkTransferNotificationTaskCompleted(context.Context, models.Tx, models.TransferID) error
	MarkTransferNotificationTaskFailed(context.Context, models.Tx, models.TransferID, string) error
}

type transfersNotificationsSender interface {
	Send(context.Context, models.Tx, models.Transfer) error
}

type Notifier struct {
	logger                       *slog.Logger
	transferNotificationsOutbox  transferNotificationsOutbox
	transfersNotificationsSender transfersNotificationsSender
	claimTasksLimit              int64
}

func NewNotifier(
	transferNotificationsOutbox transferNotificationsOutbox,
	claimTasksLimit int64,
) *Notifier {
	return &Notifier{
		transferNotificationsOutbox: transferNotificationsOutbox,
		claimTasksLimit:             claimTasksLimit,
	}
}

func (u *Notifier) SendNotifications(ctx context.Context) error {
	tx, err := u.transferNotificationsOutbox.Begin(ctx)
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

	transfers, err := u.transferNotificationsOutbox.ClaimUnsentTransferNotificationTasks(ctx, tx, u.claimTasksLimit)
	if err != nil {
		return fmt.Errorf("cannot claim unsent transfer notifications: %w", err)
	}

	var wg sync.WaitGroup
	for _, transfer := range transfers {
		wg.Go(func() {
			if err := u.transfersNotificationsSender.Send(ctx, tx, transfer); err != nil {
				markErr := u.transferNotificationsOutbox.MarkTransferNotificationTaskFailed(ctx, tx, transfer.ID, err.Error())
				if markErr != nil {
					u.logger.Error("cannot mark transfer task as failed", markErr)
				}
			} else {
				markErr := u.transferNotificationsOutbox.MarkTransferNotificationTaskCompleted(ctx, tx, transfer.ID)
				if markErr != nil {
					u.logger.Error("cannot mark transfer task as completed", markErr)
				}
			}
		})
	}

	wg.Wait()

	return nil
}
