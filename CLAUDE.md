# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Context

A payments microservice demonstrating the **Transactional Outbox Pattern** for a course work. Two implementations exist across branches:

- **`main`** — worker-based outbox (pull model): a separate `outboxrely` worker polls the outbox table and publishes to Kafka.
- **`cdc-impl`** — CDC-based outbox (push model): Debezium reads PostgreSQL WAL via logical replication and streams events to Kafka. No worker process.

The course work document is in `course_work.md`.

## Commands

```bash
# Start infrastructure (PostgreSQL, Kafka, Kafka UI, Kafka Connect)
docker compose up -d

# Run the service
go run cmd/main.go

# Apply migrations (runs automatically on startup via Goose)
# Migration files: migrations/postgres/

# Register Debezium Kafka Connect connector (cdc-impl only)
make kafka-connect-create-outbox-connector
# or:
curl -X POST localhost:8083/connectors -H "Content-Type: application/json" -d @migrations/connect/outbox.json
```

No test files exist in this project.

## Architecture

**Layered structure:**

- `cmd/main.go` — wires dependencies, starts Echo HTTP server with graceful shutdown
- `internal/domain/` — domain structs and errors (`Account`, `TransferMoneyIn`, `Tx` interface)
- `internal/usecases/money_transfer.go` — core logic: validates, locks both accounts (`FOR UPDATE`), updates balances, writes transfer + outbox event **in a single transaction**
- `internal/infra/postgres/repositories/accounts.go` — data access; `CreateMoneyTransfer()` atomically creates the transfer row and inserts the outbox event as JSON
- `internal/infra/http/server/handlers/money_transfer.go` — `POST /api/v1/accounts/transfer-money` handler

**Key outbox table columns:** `id`, `aggregate_id` (transfer UUID), `event_type` (`accounts.money.transferred`), `payload` (jsonb), `created_at`. On `main` branch there's also a `status` column (`new` → `sent`).

**Atomicity guarantee:** the use case begins a single DB transaction, updates both account balances, inserts into `transfers` and `outbox`, then commits. Event delivery is guaranteed because outbox and business state are committed together.

**CDC setup (cdc-impl):** PostgreSQL runs with `wal_level=logical`. The publication `outbox_pub` on the outbox table is created in the migration. Debezium (`migrations/connect/outbox.json`) reads the WAL slot `outbox_slot` and uses the `EventRouter` transform to route events to Kafka topics by `event_type`, keyed by `aggregate_id`.

## API

`POST /api/v1/accounts/transfer-money`
```json
{ "from_account": "<uuid>", "to_account": "<uuid>", "amount": 1 }
```
Returns `204 No Content`. Seed accounts are `11111111-...` and `22222222-...`.

## Configuration

`config.yaml` — DB connection string, HTTP address, log level. On `main` branch also has `workers.outbox_rely` with Kafka brokers, batch size, and polling interval.

## Dependencies

`pgx/v5` (Postgres), `echo/v4` (HTTP), `goose/v3` (migrations), `kafka-go` (Kafka), `google/uuid`, `yaml.v3` for config.
