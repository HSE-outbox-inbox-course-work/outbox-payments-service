package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"log/slog"
	"net"
	"strconv"
)

const migrationsPath = "migrations/postgres"

func init() {
	goose.SetLogger(
		slog.NewLogLogger(
			slog.Default().Handler(),
			slog.LevelInfo,
		),
	)
}

func Connect(
	ctx context.Context,
	connString string,
) (pool *pgxpool.Pool, err error) {
	pool, err = pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("connecting to postgres: %w", err)
	}
	defer func() {
		if err != nil && pool != nil {
			pool.Close()
		}
	}()

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pinging postgres: %w", err)
	}

	if err = applyMigrations(ctx, goose.DialectPostgres, postgresURL(pool), migrationsPath); err != nil {
		return nil, fmt.Errorf("cannot apply migrations: %w", err)
	}

	return pool, nil
}

func postgresURL(pool *pgxpool.Pool) string {
	conf := pool.Config().ConnConfig
	return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
		conf.User, conf.Password, net.JoinHostPort(conf.Host, strconv.Itoa(int(conf.Port))), conf.Database,
	)
}

func applyMigrations(
	ctx context.Context,
	dialect goose.Dialect,
	connString string,
	migrationsPath string,
) (err error) {
	slog.Info("applying migrations",
		slog.String("path", migrationsPath),
		slog.String("dialect", string(dialect)),
	)

	db, err := goose.OpenDBWithDriver(string(dialect), connString)
	if err != nil {
		return fmt.Errorf("cannot open db: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()

	if err := goose.UpContext(ctx, db, migrationsPath); err != nil {
		return fmt.Errorf("cannot up migrations: %w", err)
	}

	return nil
}
