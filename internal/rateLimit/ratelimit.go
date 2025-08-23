package rateLimit

import (
	"context"
	"time"

	redisadapter "github.com/robertarktes/ticket-reservations-and-orders/internal/adapters/redis"
)

type RateLimiter struct {
	redis *redisadapter.Cache
}

func NewRateLimiter(redis *redisadapter.Cache) *RateLimiter {
	return &RateLimiter{redis: redis}
}

func (rl *RateLimiter) Allow(ctx context.Context, key string, rate int, period time.Duration) bool {
	fullKey := "rl:" + key

	pipe := rl.redis.Client().Pipeline()
	incr := pipe.Incr(ctx, fullKey)
	pipe.Expire(ctx, fullKey, period)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false
	}

	return incr.Val() <= int64(rate)
}
