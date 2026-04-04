# Outbox Payments Service

Сервис переводов денежных средств с гарантированной доставкой событий по паттерну **Transactional Outbox**.

Эта ветка использует **воркер (polling)** для доставки событий из таблицы `outbox` в Kafka. Альтернативная реализация через CDC (Debezium) — в ветке `cdc-impl`.

## Быстрый старт

### 1. Поднять инфраструктуру

```bash
docker compose up -d
```

Запускает PostgreSQL, Kafka и Kafka UI. Топик `accounts.money.transferred` создаётся автоматически.

### 2. Запустить сервис

```bash
go run cmd/main.go
```

Сервис автоматически применит миграции и начнёт слушать на `localhost:8080`.

### 3. Запустить воркер отправки событий

```bash
go run cmd/outboxrely/main.go
```

Воркер каждую секунду опрашивает таблицу `outbox`, отправляет новые события в Kafka и помечает их как `sent`.

### 4. Сделать перевод

```bash
curl -s -X POST localhost:8080/api/v1/accounts/transfer-money \
  -H "Content-Type: application/json" \
  -d '{
    "from_account": "22222222-2222-2222-2222-222222222222",
    "to_account": "11111111-1111-1111-1111-111111111111",
    "amount": 100
  }'
```

Ответ: `204 No Content`.

### 5. Прочитать событие из Kafka

```bash
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic accounts.money.transferred \
  --from-beginning
```

Или открыть Kafka UI: http://localhost:8081

## Конфигурация воркера

В `config.yaml`:

```yaml
Workers:
  OutboxRely:
    KafkaBrokers: ["localhost:29092"]
    EventsLimit: 10
    Interval: 1s
```

## Стек

Go, PostgreSQL, Apache Kafka
