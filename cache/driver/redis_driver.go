package driver

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/arklib/ark/cache"
)

type RedisDriver struct {
	cache.Driver
	client redis.Cmdable
}

func NewRedisDriver(client redis.Cmdable) *RedisDriver {
	return &RedisDriver{client: client}
}

func (r *RedisDriver) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, data, ttl).Err()
}

func (r *RedisDriver) Get(ctx context.Context, key string) (data []byte, err error) {
	return r.client.Get(ctx, key).Bytes()
}

func (r *RedisDriver) Del(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}
