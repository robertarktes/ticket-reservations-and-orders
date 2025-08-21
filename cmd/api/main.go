package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	shutdown, err := observability.SetupOTel(context.Background(), cfg)
	if err != nil {
		log.Fatalf("failed to setup otel: %v", err)
	}
	defer shutdown()

	logger := observability.NewLogger()
	observability.InitMetrics()

	pool, err := pgxpool.New(context.Background(), cfg.CRDBDSN)
	if err != nil {
		log.Fatalf("failed to connect to crdb: %v", err)
	}
	defer pool.Close()
	crdbRepo := crdb.NewRepository(pool)

	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("failed to connect to mongo: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())
	mongoDB := mongoClient.Database("tro")
	mongoCatalog := mongoadapter.NewCatalogRepository(mongoDB, logger)

	redisClient := redisclient.NewClient(&redisclient.Options{Addr: cfg.RedisAddr})
	redisCache := redisadapter.NewCache(redisClient)
	redisIdemp := redisadapter.NewIdempotency(redisClient)
	idemp := idempotency.NewIdempotency(redisIdemp, time.Hour)
	rl := rateLimit.NewRateLimiter(redisCache)

	rabbitConn, err := amqp.Dial(cfg.RabbitURL)
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}
	defer rabbitConn.Close()
	rabbitPub, err := rabbit.NewPublisher(rabbitConn)
	if err != nil {
		log.Fatalf("failed to create publisher: %v", err)
	}

	handlers := httphandler.NewHandlers(cfg, crdbRepo, redisCache, idemp, mongoCatalog, rabbitPub)

	r := httphandler.SetupRouter(handlers, logger, rl, idemp)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	logger.Info("Server exiting")
}
