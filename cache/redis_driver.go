package cache

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

func (r *RedisDriver) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisDriver) Get(ctx context.Context, key string) (value []byte, err error) {
	return r.client.Get(ctx, key).Bytes()
}

func (r *RedisDriver) Del(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}
