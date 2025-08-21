package idempotency

import (
	"context"
	"time"

	redisadapter "github.com/robertarktes/ticket-reservations-and-orders/internal/adapters/redis"
)

type Idempotency struct {
	redis *redisadapter.Idempotency
	ttl   time.Duration
}

func NewIdempotency(redis *redisadapter.Idempotency, ttl time.Duration) *Idempotency {
	return &Idempotency{redis: redis, ttl: ttl}
}

type Response struct {
	Status int
	Result []byte
}

func (i *Idempotency) Get(ctx context.Context, key string) (*Response, error) {

	return nil, nil
}

func (i *Idempotency) Set(ctx context.Context, key string, resp Response) error {

	return nil
}
