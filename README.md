# Ticket Reservations and Orders

Микросервисная система для бронирования и заказа билетов.

## Архитектура

- **API Gateway** - HTTP API для клиентов
- **Expiry Worker** - обработка истекающих бронирований
- **Outbox Publisher** - публикация событий в очередь

## Технологии

- Go 1.21+
- CockroachDB (PostgreSQL)
- Redis
- MongoDB
- RabbitMQ
- OpenTelemetry

## Запуск

```bash
# Зависимости
go mod tidy

# Сборка
go build ./...

# Запуск API
go run cmd/api/main.go

# Запуск worker'ов
go run cmd/expiry-worker/main.go
go run cmd/outbox-publisher/main.go
```

## Docker

```bash
docker-compose up -d
```

## Тесты

```bash
go test ./...
```
