package job

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
)

type RedisDriver struct {
	Driver
	client redis.Cmdable
}

func NewRedisDriver(client redis.Cmdable) *RedisDriver {
	return &RedisDriver{client: client}
}

func (r *RedisDriver) Push(ctx context.Context, queue string, data []byte) error {
	return r.client.LPush(ctx, queue, data).Err()
}

func (r *RedisDriver) Pop(ctx context.Context, queue string) ([]byte, error) {
	result, err := r.client.LPop(ctx, queue).Bytes()
	if errors.Is(err, redis.Nil) {
		return result, nil
	}
	return result, err
}
