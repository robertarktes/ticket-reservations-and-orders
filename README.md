# Ticket Reservations & Orders (TRO)

A production-ready microservices system for ticket booking and order management with event sourcing, built with Go.

## 🏗️ Architecture

- **API Gateway** - RESTful HTTP API with idempotency and rate limiting
- **Expiry Worker** - Background service for handling expired reservations
- **Outbox Publisher** - Event sourcing pattern for reliable message publishing
- **Event Store** - CockroachDB with outbox pattern for consistency

## 🚀 Features

- **Idempotency** - Prevents duplicate operations with Redis-based caching
- **Rate Limiting** - Sliding window rate limiting per user/IP
- **Event Sourcing** - Reliable event publishing via outbox pattern
- **Distributed Transactions** - Serializable isolation level for consistency
- **Observability** - OpenTelemetry integration with structured logging
- **Graceful Shutdown** - Proper cleanup and connection management

## 🛠️ Tech Stack

- **Language**: Go 1.21+
- **Database**: CockroachDB (PostgreSQL-compatible)
- **Cache**: Redis for idempotency and rate limiting
- **Message Queue**: RabbitMQ for event publishing
- **Document Store**: MongoDB for event catalog
- **Observability**: OpenTelemetry, structured logging
- **Build**: Go modules with fast, cross-platform compilation

## 📋 Prerequisites

- Go 1.21+
- Go 1.21+ (for modern features)
- Docker & Docker Compose
- CockroachDB
- Redis
- RabbitMQ
- MongoDB

## 🚀 Quick Start

```bash
# Clone and setup
git clone https://github.com/robertarktes/ticket-reservations-and-orders.git
cd ticket-reservations-and-orders

# Install dependencies
go mod tidy

# Build with Go (recommended for development)
go build ./...

# Build with Bazel (requires external deps setup - see WORKSPACE)
# bazel build --enable_workspace --noenable_bzlmod //cmd/api:api

# Run with Docker
cd deploy
docker-compose up -d

# Check services
docker-compose ps

# Start services
go run cmd/api/main.go &
go run cmd/expiry-worker/main.go &
go run cmd/outbox-publisher/main.go &
```

## 📚 API Endpoints

### Holds
- `POST /v1/holds` - Create seat reservation
- `GET /v1/holds/{id}` - Get reservation details

### Orders
- `POST /v1/orders` - Create order from reservation
- `GET /v1/orders/{id}` - Get order details
- `POST /v1/payments/callback` - Payment confirmation

### Health
- `GET /v1/healthz` - Health check
- `GET /v1/readyz` - Readiness check
- `GET /metrics` - Prometheus metrics

## 🔧 Configuration

Environment variables:
```bash
CRDB_DSN=postgresql://user:pass@localhost:26257/tro
MONGO_URI=mongodb://localhost:27017
REDIS_ADDR=localhost:6379
RABBIT_URL=amqp://guest:guest@localhost:5672/
HOLD_TTL=5m
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
```

## 🧪 Testing

```bash
# Test with Go
go test ./internal/...

# Integration tests (requires Docker)
cd deploy
docker-compose up -d
cd ..
go test ./test/integration/...
cd deploy
docker-compose down
cd ..
```

## 🔧 Build Features

- **Go Modules**: Modern dependency management
- **Bazel Support**: Fast, hermetic builds (see WORKSPACE for setup)
- **Cross-Platform**: Build for any OS/architecture
- **Docker Integration**: Containerized builds

```bash
# Build for specific platform
GOOS=linux GOARCH=amd64 go build ./...

# Build with optimizations
go build -ldflags="-s -w" ./...

# Build with race detection
go build -race ./...

# Bazel builds (requires external deps configuration)
# bazel build --enable_workspace --noenable_bzlmod //cmd/api:api

# For Bazel setup, configure external dependencies in WORKSPACE
# Example: go_repository for external Go modules
```

## 📊 Monitoring

- **Metrics**: Prometheus endpoints
- **Tracing**: OpenTelemetry with Jaeger
- **Logging**: Structured JSON logging
- **Health**: Database and service health checks

## 🏭 Production Features

- **Circuit Breaker**: Protection against cascading failures
- **Retry Logic**: Exponential backoff for external services
- **Graceful Shutdown**: Proper cleanup on termination
- **Connection Pooling**: Efficient database connections
- **Transaction Management**: Serializable isolation for consistency

## 🔒 Security

- JWT authentication middleware
- Rate limiting per user/IP
- Input validation and sanitization
- Secure headers and CORS configuration

## 📈 Performance

- **Connection Pooling**: Efficient database connections
- **Redis Caching**: Fast idempotency checks
- **Batch Processing**: Efficient outbox publishing
- **Async Workers**: Non-blocking background processing

## 🚀 Deployment

```bash
# Build binaries with Go
go build -o bin/api cmd/api/main.go
go build -o bin/expiry-worker cmd/expiry-worker/main.go
go build -o bin/outbox-publisher cmd/outbox-publisher/main.go

# Build Docker images
docker build -t tro-api cmd/api/
docker build -t tro-worker cmd/expiry-worker/
docker build -t tro-publisher cmd/outbox-publisher/

# Deploy with Kubernetes
kubectl apply -f deploy/
```

## 🤝 Contributing

1. Fork the repository
2. Create feature branch
3. Commit changes
4. Push to branch
5. Create Pull Request

## 📄 License

MIT License - see LICENSE file for details
