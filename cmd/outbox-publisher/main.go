package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/adapters/crdb"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/adapters/rabbit"
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

	conn, err := amqp.Dial(cfg.RabbitURL)
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}
	defer conn.Close()
	rabbitPub, err := rabbit.NewPublisher(conn)
	if err != nil {
		log.Fatalf("failed to create publisher: %v", err)
	}

	publisher := NewOutboxPublisher(repo, rabbitPub, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go publisher.Run(ctx)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logger.Info("Shutdown outbox publisher")
}

type OutboxPublisher struct {
	repo      *crdb.Repository
	rabbitPub *rabbit.Publisher
	logger    observability.Logger
}

func NewOutboxPublisher(repo *crdb.Repository, rabbitPub *rabbit.Publisher, logger observability.Logger) *OutboxPublisher {
	return &OutboxPublisher{repo: repo, rabbitPub: rabbitPub, logger: logger}
}

func (p *OutboxPublisher) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			records, err := p.repo.GetUnpublishedOutbox(ctx, 100)
			if err != nil {
				p.logger.Error("failed to get outbox records", err)
				continue
			}
			for _, rec := range records {
				if err := p.publishWithRetry(ctx, rec); err != nil {
					p.logger.Error("failed to publish after retries", err)
				}
			}
		}
	}
}

func (p *OutboxPublisher) publishWithRetry(ctx context.Context, rec crdb.OutboxRecord) error {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		msg := amqp.Publishing{
			MessageId:   rec.DedupeKey,
			ContentType: "application/json",
			Body:        rec.Payload,
		}

		err := p.rabbitPub.Publish(ctx, rec.EventType, msg)
		if err == nil {
			return p.repo.MarkPublished(ctx, rec.ID, time.Now(), rec.DedupeKey)
		}

		backoff := time.Duration(1<<i) * time.Second
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}
	return fmt.Errorf("failed after %d retries", maxRetries)
}
