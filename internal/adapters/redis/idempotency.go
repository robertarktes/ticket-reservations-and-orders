package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type Idempotency struct {
	client *redis.Client
}

func NewIdempotency(client *redis.Client) *Idempotency {
	return &Idempotency{client: client}
}

type IdempResponse struct {
	Status int
	Result []byte
}

func (i *Idempotency) Get(ctx context.Context, key string) (*IdempResponse, error) {
	val, err := i.client.Get(ctx, "idemp:"+key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var resp IdempResponse
	err = json.Unmarshal(val, &resp)
	return &resp, err
}

func (i *Idempotency) Set(ctx context.Context, key string, resp IdempResponse, ttl time.Duration) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return i.client.Set(ctx, "idemp:"+key, data, ttl).Err()
}
