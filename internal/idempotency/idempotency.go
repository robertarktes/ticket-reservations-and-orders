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
	idempResp, err := i.redis.Get(ctx, key)
	if err != nil || idempResp == nil {
		return nil, err
	}

	return &Response{
		Status: idempResp.Status,
		Result: idempResp.Result,
	}, nil
}

func (i *Idempotency) Set(ctx context.Context, key string, resp Response) error {
	idempResp := redisadapter.IdempResponse{
		Status: resp.Status,
		Result: resp.Result,
	}
	return i.redis.Set(ctx, key, idempResp, i.ttl)
}
