package queue

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisDriver struct {
	Driver
	client redis.Cmdable
}

func NewRedisDriver(client redis.Cmdable) *RedisDriver {
	return &RedisDriver{client: client}
}

func (r *RedisDriver) Produce(ctx context.Context, topic string, message []byte) error {
	args := &redis.XAddArgs{
		Stream: topic,
		Values: map[string]any{"message": string(message)},
	}
	return r.client.XAdd(ctx, args).Err()
}

func (r *RedisDriver) Consume(ctx context.Context, topic, group string, handler ConsumeTaskHandler) error {
	err := r.InitGroup(ctx, topic, group)
	if err != nil {
		return err
	}

	u, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	args := &redis.XReadGroupArgs{
		Group:    group,
		Consumer: u.String(),
		Streams:  []string{topic, ">"},
		Block:    0,
		// Count:    1,
	}
	for {
		streams, err := r.client.XReadGroup(ctx, args).Result()
		if err != nil {
			fmt.Printf("[redis.xRead] topic: %s, group: %s, error: %v\n", topic, group, err)
			time.Sleep(time.Second)
			continue
		}

		stream := streams[0]
		for _, message := range stream.Messages {
			rawMessage, ok := message.Values["message"]
			if !ok {
				continue
			}

			err = handler([]byte(rawMessage.(string)))
			if err != nil {
				continue
			}

			_, err = r.client.XAck(ctx, stream.Stream, group, message.ID).Result()
			if err != nil {
				log.Printf("[redis.xAck] topic: %s, group: %s, messageId: %s, error: %v\n",
					topic, group, message.ID, err)
				continue
			}
		}
	}
}

func (r *RedisDriver) InitGroup(ctx context.Context, topic, name string) error {
	groups, err := r.client.XInfoGroups(ctx, topic).Result()
	if err != nil {
		return err
	}

	for _, group := range groups {
		if name == group.Name {
			return nil
		}
	}
	return r.client.XGroupCreateMkStream(ctx, topic, name, "0").Err()
}
