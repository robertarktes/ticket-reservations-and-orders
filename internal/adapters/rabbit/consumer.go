package rabbit

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	ch *amqp.Channel
}

func NewConsumer(conn *amqp.Connection, queue string) (*Consumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	_, err = ch.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	return &Consumer{ch: ch}, nil
}

func (c *Consumer) Consume(ctx context.Context) (<-chan amqp.Delivery, error) {
	return c.ch.Consume("billing.q", "", false, false, false, false, nil)
}
