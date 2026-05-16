FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o outbox ./cmd

FROM alpine:3.21
WORKDIR /app
COPY --from=builder /app/outbox .
COPY config.yaml .
COPY migrations/ migrations/
ENTRYPOINT ["/app/outbox"]
