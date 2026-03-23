# outbox-payments-service

## Запуск
- docker compose up -d
- go run cmd/main.go (применение миграций, иначе connect сам создаст publication)
- make kafka-connect-create-outbox-connector
- можно делать запросы и смотреть в кафку

outbox table должна выглядеть так

https://debezium.io/documentation/reference/stable/transformations/outbox-event-router.html

```json
{
  "name": "outbox-connector",
  "config": {
    "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
    "database.hostname": "postgres",
    "database.port": "5432",
    "database.user": "admin",
    "database.password": "password",
    "database.dbname": "payments-service-db",
    "topic.prefix": "payments-service-db",
    "table.include.list": "public.outbox",
    "plugin.name": "pgoutput",
    "publication.name": "outbox_pub",
    "tombstones.on.delete": "false",
    "slot.name": "outbox_slot",
    "key.converter": "org.apache.kafka.connect.json.JsonConverter",
    "value.converter": "org.apache.kafka.connect.json.JsonConverter",
    "transforms": "outbox", // явно задаем плагин
    "transforms.outbox.type": "io.debezium.transforms.outbox.EventRouter",
    "transforms.outbox.route.by.field": "event_type",
    "transforms.outbox.route.topic.replacement": "${routedByValue}",
    "transforms.outbox.table.field.event.key": "aggregate_id"
  }
}
```