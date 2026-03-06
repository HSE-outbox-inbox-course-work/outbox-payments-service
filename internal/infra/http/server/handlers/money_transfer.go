package handlers

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"net/http"
	"outbox-payment-service/internal/domain"

	"github.com/labstack/echo/v4"
)

type transferMoneyRequest struct {
	FromAccount uuid.UUID `json:"from_account"`
	ToAccount   uuid.UUID `json:"to_account"`
	Amount      int64     `json:"amount"`
}

type moneyTransfer interface {
	TransferMoney(ctx context.Context, in *domain.TransferMoneyIn) (err error)
}

type MoneyTransfer struct {
	moneyTransfer moneyTransfer
}

func NewMoneyTransfer(moneyTransfer moneyTransfer) *MoneyTransfer {
	return &MoneyTransfer{
		moneyTransfer: moneyTransfer,
	}
}

func (u *MoneyTransfer) ServeHTTP(c echo.Context) error {
	var req transferMoneyRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	err := u.moneyTransfer.TransferMoney(c.Request().Context(), &domain.TransferMoneyIn{
		FromAccount: domain.AccountID(req.FromAccount),
		ToAccount:   domain.AccountID(req.ToAccount),
		Amount:      req.Amount,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrAccountNotFound):
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrInsufficientFunds):
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		case errors.Is(err, domain.ErrInvalidMoneyTransferAmount):
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, echo.ErrInternalServerError.Error())
		}
	}

	return c.NoContent(http.StatusNoContent)
}
