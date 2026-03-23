kafka-connect-create-outbox-connector:
	curl -X POST localhost:8083/connectors -H "Content-Type: application/json" -d @migrations/connect/outbox.json