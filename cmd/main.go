package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"outbox-payment-service/internal/infra/http/server/handlers"
	"outbox-payment-service/internal/infra/postgres/repositories"
	"outbox-payment-service/internal/usecases"
	"outbox-payment-service/pkg/sl"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	postgresConn, err := pgxpool.New(ctx, os.Getenv("APP_POSTGRES_CONN_STRING"))
	if err != nil {
		logger.Error("cannot connect to postgres", sl.Error(err))
		return
	}

	accountsRepository := repositories.NewAccounts(postgresConn)

	moneyTransferUseCase := usecases.NewMoneyTransfer(accountsRepository)

	moneyTransferHandler := handlers.NewMoneyTransfer(moneyTransferUseCase)

	echoServer := echo.New()
	defer func() {
		if err := echoServer.Shutdown(context.TODO()); err != nil {
			logger.Error("cannot shutdown echo server", sl.Error(err))
		}
	}()

	echoServer.POST("/api/v1/accounts/transfer-money", moneyTransferHandler.ServeHTTP)

	go func() {
		err := echoServer.Start(":8080")
		if errors.Is(err, http.ErrServerClosed) {
			logger.Error("echo server has been shutdown")
			cancel()
		}
		if err != nil {
			logger.Error("cannot start echo server", sl.Error(err))
			cancel()
		}
	}()

	logger.Info("http server started")

	logger.Info("shutting down application")
	<-ctx.Done()
	logger.Info("application shutdown complete")
}
