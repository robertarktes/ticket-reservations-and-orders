package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	redisclient "github.com/redis/go-redis/v9"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/adapters/crdb"
	mongoadapter "github.com/robertarktes/ticket-reservations-and-orders/internal/adapters/mongo"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/adapters/rabbit"
	redisadapter "github.com/robertarktes/ticket-reservations-and-orders/internal/adapters/redis"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/config"
	httphandler "github.com/robertarktes/ticket-reservations-and-orders/internal/http"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/idempotency"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/observability"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/rateLimit"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestIntegration_HoldOrderConfirm(t *testing.T) {
	ctx := context.Background()

	crdbContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "cockroachdb/cockroach:v24.1.1",
			Cmd:          []string{"start-single-node", "--insecure"},
			ExposedPorts: []string{"26257/tcp", "8080/tcp"},
			WaitingFor:   wait.ForHTTP("/health?ready=1").WithPort("8080"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer crdbContainer.Terminate(ctx)

	mongoContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "mongo:7",
			ExposedPorts: []string{"27017/tcp"},
			WaitingFor:   wait.ForExec([]string{"mongo", "--eval", "db.runCommand('ping').ok"}),
		},
		Started: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer mongoContainer.Terminate(ctx)

	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForExec([]string{"redis-cli", "ping"}),
		},
		Started: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer redisContainer.Terminate(ctx)

	rabbitContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "rabbitmq:3.13-management",
			ExposedPorts: []string{"5672/tcp", "15672/tcp"},
			WaitingFor:   wait.ForHTTP("/api/health").WithPort("15672"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer rabbitContainer.Terminate(ctx)

	crdbHost, err := crdbContainer.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	crdbPort, err := crdbContainer.MappedPort(ctx, "26257")
	if err != nil {
		t.Fatal(err)
	}
	mongoHost, err := mongoContainer.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	mongoPort, err := mongoContainer.MappedPort(ctx, "27017")
	if err != nil {
		t.Fatal(err)
	}
	redisHost, err := redisContainer.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	redisPort, err := redisContainer.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatal(err)
	}
	rabbitHost, err := rabbitContainer.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	rabbitPort, err := rabbitContainer.MappedPort(ctx, "5672")
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		CRDBDSN:      "postgresql://root@" + crdbHost + ":" + crdbPort.Port() + "/tro?sslmode=disable",
		MongoURI:     "mongodb://" + mongoHost + ":" + mongoPort.Port(),
		RedisAddr:    redisHost + ":" + redisPort.Port(),
		RabbitURL:    "amqp://guest:guest@" + rabbitHost + ":" + rabbitPort.Port() + "/",
		HoldTTL:      300 * time.Second,
		OTLPEndpoint: "", // Skip otel for test
	}

	// Setup dependencies
	pool, err := pgxpool.New(ctx, cfg.CRDBDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	crdbRepo := crdb.NewRepository(pool)

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		t.Fatal(err)
	}
	defer mongoClient.Disconnect(ctx)
	mongoDB := mongoClient.Database("tro")
	logger := observability.NewLogger()
	mongoCatalog := mongoadapter.NewCatalogRepository(mongoDB, logger)

	redisClient := redisclient.NewClient(&redisclient.Options{Addr: cfg.RedisAddr})
	redisCache := redisadapter.NewCache(redisClient)
	redisIdemp := redisadapter.NewIdempotency(redisClient)
	idemp := idempotency.NewIdempotency(redisIdemp, time.Hour)
	rl := rateLimit.NewRateLimiter(redisCache)

	rabbitConn, err := amqp.Dial(cfg.RabbitURL)
	if err != nil {
		t.Fatal(err)
	}
	defer rabbitConn.Close()
	rabbitPub, err := rabbit.NewPublisher(rabbitConn)
	if err != nil {
		t.Fatal(err)
	}

	handlers := httphandler.NewHandlers(cfg, crdbRepo, redisCache, idemp, mongoCatalog, rabbitPub)
	r := httphandler.SetupRouter(handlers, logger, rl, idemp)

	// Start server
	srv := &http.Server{Addr: ":8080", Handler: r}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Fatal(err)
		}
	}()
	defer srv.Shutdown(ctx)

	// Test scenario
	eventID := uuid.New()
	userID := uuid.New()

	event := mongoadapter.EventDoc{
		ID:   eventID,
		Name: "Test Event",
		Seats: []mongoadapter.SeatDoc{
			{Number: "A1", Row: "A", Section: "Main", Price: 100.0, Available: true},
		},
	}
	err = mongoCatalog.CreateEvent(ctx, event)
	if err != nil {
		t.Fatal(err)
	}

	// Test hold
	holdReq := map[string]interface{}{
		"event_id": eventID.String(),
		"seats":    []string{"A1"},
		"user_id":  userID.String(),
	}
	holdBody, _ := json.Marshal(holdReq)
	req, _ := http.NewRequest("POST", "http://localhost:8080/v1/holds", bytes.NewReader(holdBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", uuid.New().String())
	req.Header.Set("Authorization", "Bearer mock")
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("hold failed: %v, status: %d", err, resp.StatusCode)
	}

	// Test order
	orderReq := map[string]interface{}{
		"event_id":       eventID.String(),
		"seats":          []string{"A1"},
		"user_id":        userID.String(),
		"payment_method": "card",
	}
	orderBody, _ := json.Marshal(orderReq)
	req, _ = http.NewRequest("POST", "http://localhost:8080/v1/orders", bytes.NewReader(orderBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", uuid.New().String())
	req.Header.Set("Authorization", "Bearer mock")
	resp, err = http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusAccepted {
		t.Fatalf("order failed: %v, status: %d", err, resp.StatusCode)
	}

	var orderResp struct {
		OrderID uuid.UUID `json:"order_id"`
	}
	json.NewDecoder(resp.Body).Decode(&orderResp)

	// Test payment
	paymentReq := map[string]interface{}{
		"order_id":       orderResp.OrderID.String(),
		"status":         "SUCCEEDED",
		"transaction_id": "tx123",
	}
	paymentBody, _ := json.Marshal(paymentReq)
	req, _ = http.NewRequest("POST", "http://localhost:8080/v1/payments/callback", bytes.NewReader(paymentBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("payment failed: %v, status: %d", err, resp.StatusCode)
	}

	// Verify order status
	req, _ = http.NewRequest("GET", "http://localhost:8080/v1/orders/"+orderResp.OrderID.String(), nil)
	req.Header.Set("Authorization", "Bearer mock")
	resp, err = http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("get order failed: %v, status: %d", err, resp.StatusCode)
	}
	var getOrderResp struct {
		Status string `json:"status"`
	}
	json.NewDecoder(resp.Body).Decode(&getOrderResp)
	if getOrderResp.Status != "CONFIRMED" {
		t.Errorf("expected status CONFIRMED, got %s", getOrderResp.Status)
	}
}
