package rabbit

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	ch *amqp.Channel
}

func NewPublisher(conn *amqp.Connection) (*Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	err = ch.ExchangeDeclare("tro.events", "topic", true, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	return &Publisher{ch: ch}, nil
}

func (p *Publisher) Publish(ctx context.Context, key string, msg amqp.Publishing) error {
	return p.ch.PublishWithContext(ctx, "tro.events", key, false, false, msg)
}
