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

	return true
}
