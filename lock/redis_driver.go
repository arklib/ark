package lock

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisDriver struct {
	Driver
	client redis.Cmdable
}

func NewRedisDriver(client redis.Cmdable) *RedisDriver {
	return &RedisDriver{client: client}
}

func (r *RedisDriver) Lock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return r.client.SetNX(ctx, key, 1, ttl).Result()
}

func (r *RedisDriver) Free(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}
