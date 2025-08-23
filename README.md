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
- **Build**: Bazel for fast, reproducible, incremental builds

## 📋 Prerequisites

- Go 1.21+
- Bazel 6.0+ (for advanced builds)
- Docker & Docker Compose
- CockroachDB
- Redis
- RabbitMQ
- MongoDB

## 🚀 Quick Start

```bash
# Clone and setup
git clone <repository>
cd ticket-reservations-and-orders

# Install dependencies
go mod tidy

# Build with Bazel
bazel build //cmd/api:api
bazel build //cmd/expiry-worker:expiry_worker
bazel build //cmd/outbox-publisher:outbox_publisher

# Or build with Go
go build ./...

# Run with Docker
docker-compose up -d

# Start services
bazel run //cmd/api:api &
bazel run //cmd/expiry-worker:expiry_worker &
bazel run //cmd/outbox-publisher:outbox_publisher &
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
# Build and test with Bazel
bazel build //...
bazel test //...

# Or test with Go
go test ./...

# Integration tests
go test ./test/integration/...
```

## 🔧 Bazel Features

- **Incremental Builds**: Only rebuilds changed dependencies
- **Remote Caching**: Shared build cache across team
- **Hermetic Builds**: Reproducible builds with locked toolchains
- **Parallel Execution**: Fast builds with dependency graph optimization

```bash
# Build specific targets
bazel build //cmd/api:api
bazel build //internal/...

# Run tests with caching
bazel test //... --test_output=errors

# Query dependency graph
bazel query "deps(//cmd/api:api)"

# Build with remote cache (if configured)
bazel build //... --remote_cache=grpc://cache.example.com:9092
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
# Build binaries with Bazel
bazel build //cmd/api:api //cmd/expiry-worker:expiry_worker //cmd/outbox-publisher:outbox_publisher

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
