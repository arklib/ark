package lock

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	Driver
	client redis.Cmdable
}

func NewRedis(client redis.Cmdable) *Redis {
	return &Redis{client: client}
}

func (r *Redis) Set(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	return r.client.SetNX(ctx, key, value, ttl).Result()
}

func (r *Redis) Del(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}
