package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

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
	p.logger.Info("Outbox publisher started")
	<-ctx.Done()
}
