package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	redisclient "github.com/redis/go-redis/v9"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/adapters/crdb"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/adapters/rabbit"
	redisadapter "github.com/robertarktes/ticket-reservations-and-orders/internal/adapters/redis"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/config"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/observability"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	shutdownOtel, err := observability.SetupOTel(context.Background(), cfg)
	if err != nil {
		log.Fatalf("failed to setup otel: %v", err)
	}
	defer shutdownOtel()

	logger := observability.NewLogger()

	pool, err := pgxpool.New(context.Background(), cfg.CRDBDSN)
	if err != nil {
		log.Fatalf("failed to connect to crdb: %v", err)
	}
	defer pool.Close()
	repo := crdb.NewRepository(pool)

	redisClient := redisclient.NewClient(&redisclient.Options{Addr: cfg.RedisAddr})
	redisCache := redisadapter.NewCache(redisClient)

	conn, err := amqp.Dial(cfg.RabbitURL)
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}
	defer conn.Close()
	rabbitPub, err := rabbit.NewPublisher(conn)
	if err != nil {
		log.Fatalf("failed to create publisher: %v", err)
	}

	worker := NewExpiryWorker(repo, redisCache, rabbitPub, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go worker.Run(ctx, time.Minute)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logger.Info("Shutdown expiry worker")
}

type ExpiryWorker struct {
	repo      *crdb.Repository
	redis     *redisadapter.Cache
	rabbitPub *rabbit.Publisher
	logger    observability.Logger
}

func NewExpiryWorker(repo *crdb.Repository, redis *redisadapter.Cache, rabbitPub *rabbit.Publisher, logger observability.Logger) *ExpiryWorker {
	return &ExpiryWorker{repo: repo, redis: redis, rabbitPub: rabbitPub, logger: logger}
}

func (w *ExpiryWorker) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			holds, err := w.repo.GetExpiredHolds(ctx, now)
			if err != nil {
				w.logger.Error("failed to get expired holds", err)
				continue
			}
			for _, hold := range holds {
				err := w.repo.ReleaseHold(ctx, hold.ID)
				if err != nil {
					w.logger.Error("failed to release hold", err)
					continue
				}
				for _, seat := range hold.Seats {
					redisClient := redisclient.NewClient(&redisclient.Options{Addr: "localhost:6379"})
					redisClient.Del(ctx, "hold:"+hold.EventID.String()+":"+seat)
				}
				payload, _ := json.Marshal(map[string]interface{}{"hold_id": hold.ID})
				msg := amqp.Publishing{
					MessageId:   uuid.New().String(),
					ContentType: "application/json",
					Body:        payload,
				}
				w.rabbitPub.Publish(ctx, "hold.expired", msg)
			}
		}
	}
}
