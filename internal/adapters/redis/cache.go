package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
}

func NewCache(client *redis.Client) *Cache {
	return &Cache{client: client}
}

func (c *Cache) Client() *redis.Client {
	return c.client
}

func (c *Cache) SetHoldLock(ctx context.Context, eventID, seat string, userID string, ttl time.Duration) (bool, error) {
	key := "hold:" + eventID + ":" + seat
	res := c.client.SetNX(ctx, key, userID, ttl)
	return res.Val(), res.Err()
}
