package main

import (
	"context"
	"github.com/segmentio/kafka-go"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"outbox-payment-service/internal/config"
	"outbox-payment-service/internal/infra/postgres"
	"outbox-payment-service/internal/infra/postgres/repositories"
	"outbox-payment-service/internal/workers/outboxrely"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	conf, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	postgresConn, err := postgres.Connect(ctx, conf.Postgres.ConnString)
	if err != nil {
		log.Fatal(err)
	}

	outboxRepo := repositories.NewOutbox(postgresConn)

	kafkaWriter := &kafka.Writer{
		Addr:         kafka.TCP(conf.Workers.OutboxRelyWorker.KafkaBrokers...),
		Balancer:     new(kafka.Hash),
		RequiredAcks: kafka.RequireAll,
	}

	worker := outboxrely.NewWorker(
		outboxRepo,
		kafkaWriter,
		conf.Workers.OutboxRelyWorker.EventsLimit,
		conf.Workers.OutboxRelyWorker.Interval,
	)

	go worker.Run(ctx)
	slog.Info("worker started")

	<-ctx.Done()
	slog.Info("shutting down")
}
