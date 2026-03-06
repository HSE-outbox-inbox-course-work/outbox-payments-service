package main

import (
	"context"
	"errors"
	"github.com/labstack/echo/v4/middleware"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"outbox-payment-service/internal/config"
	"outbox-payment-service/internal/infra/http/server/handlers"
	"outbox-payment-service/internal/infra/postgres"
	"outbox-payment-service/internal/infra/postgres/repositories"
	"outbox-payment-service/internal/usecases"
	"outbox-payment-service/pkg/sl"
	"syscall"

	"github.com/labstack/echo/v4"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	conf, err := config.Read()
	if err != nil {
		log.Panicf("error reading config: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.Level(conf.Logger.Level),
	}))

	slog.SetDefault(logger)

	postgresConn, err := postgres.Connect(ctx, conf.Postgres.ConnString)
	if err != nil {
		logger.Error("cannot connect to postgres", sl.Error(err))
		return
	}

	accountsRepository := repositories.NewAccounts(postgresConn)

	moneyTransferUseCase := usecases.NewMoneyTransfer(accountsRepository)

	moneyTransferHandler := handlers.NewMoneyTransfer(moneyTransferUseCase)

	echoServer := echo.New()

	echoServer.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod: true,
		LogURI:    true,
		LogStatus: true,
		LogError:  true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			attrs := []slog.Attr{
				slog.String("method", v.Method),
				slog.String("uri", v.URI),
				slog.Int("status", v.Status),
				slog.String("remote_ip", c.RealIP()),
			}

			if v.Error != nil {
				attrs = append(attrs, sl.Error(v.Error))
			}

			var level slog.Level
			if v.Error != nil {
				level = slog.LevelError
			} else {
				level = slog.LevelInfo
			}

			slog.LogAttrs(c.Request().Context(), level, "http request", attrs...)
			return nil
		},
	}))

	echoServer.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		StackSize: 1 << 10,
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			slog.Error("panic recovered", sl.Error(err),
				slog.String("stack", string(stack)),
			)
			return err
		},
	}))

	defer func() {
		if err := echoServer.Shutdown(context.TODO()); err != nil {
			slog.Error("cannot shutdown echo server", sl.Error(err))
		}
	}()

	echoServer.POST("/api/v1/accounts/transfer-money", moneyTransferHandler.ServeHTTP)

	go func() {
		err := echoServer.Start(conf.HTTPServer.Address)
		if errors.Is(err, http.ErrServerClosed) {
			slog.Error("echo server has been shutdown")
			cancel()
		}
		if err != nil {
			slog.Error("cannot start echo server", sl.Error(err))
			cancel()
		}
	}()

	slog.Info("http server started")

	slog.Info("app started")
	<-ctx.Done()
	slog.Info("shutting down application")
}
