# Outbox Payments Service

Сервис переводов денежных средств с гарантированной доставкой событий по паттерну **Transactional Outbox**.

Эта ветка использует **CDC (Change Data Capture)** через Debezium для доставки событий из таблицы `outbox` в Kafka. Альтернативная реализация с воркером — в ветке `worker-impl`.

## Быстрый старт

### 1. Поднять инфраструктуру

```bash
docker compose up -d
```

Запускает PostgreSQL (`wal_level=logical`), Kafka, Kafka UI и Kafka Connect (Debezium).

### 2. Запустить сервис

```bash
go run cmd/main.go
```

Сервис автоматически применит миграции и начнёт слушать на `localhost:8080`.

> Важно запустить сервис **до** регистрации коннектора — миграции создают таблицу `outbox` и publication `outbox_pub`, которые нужны Debezium.

### 3. Зарегистрировать Debezium-коннектор

```bash
make kafka-connect-create-outbox-connector
```

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

## Стек

Go, PostgreSQL, Apache Kafka, Debezium, Kafka Connect
