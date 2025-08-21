package outbox

import (
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/adapters/crdb"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/adapters/rabbit"
)

type Publisher struct {
	repo      *crdb.Repository
	rabbitPub *rabbit.Publisher
}

func NewPublisher(repo *crdb.Repository, rabbitPub *rabbit.Publisher) *Publisher {
	return &Publisher{repo: repo, rabbitPub: rabbitPub}
}

func (p *Publisher) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			records, err := p.repo.GetUnpublishedOutbox(ctx, 10)
			if err != nil {
				// log
				continue
			}
			for _, rec := range records {
				msg := amqp.Publishing{
					MessageId:   rec.DedupeKey,
					ContentType: "application/json",
					Body:        rec.Payload,
				}
				err := p.rabbitPub.Publish(ctx, rec.EventType, msg)
				if err != nil {
					continue
				}
				p.repo.MarkPublished(ctx, rec.ID, time.Now(), rec.DedupeKey)
			}
		}
	}
}
