package driver

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"

	"github.com/arklib/ark/queue"
)

type RedisDriver struct {
	queue.Driver
	ttl    int64
	maxLen int64
	client redis.Cmdable
}

func NewRedisDriver(client redis.Cmdable) *RedisDriver {
	return &RedisDriver{client: client}
}

func (r *RedisDriver) New() *RedisDriver {
	return &RedisDriver{
		ttl:    r.ttl,
		maxLen: r.maxLen,
		client: r.client,
	}
}

func (r *RedisDriver) WithTTL(ttl int64) *RedisDriver {
	r.ttl = ttl
	return r
}

func (r *RedisDriver) WithMaxLen(len int64) *RedisDriver {
	r.maxLen = len
	return r
}

func (r *RedisDriver) Produce(ctx context.Context, topic string, message []byte) error {
	args := &redis.XAddArgs{
		Stream: topic,
		Values: map[string]any{"message": string(message)},
		Approx: true,
	}

	// auto clear message
	switch {
	case r.maxLen > 0:
		args.MaxLen = r.maxLen
	case r.ttl > 0:
		expired := time.Now().Unix() - r.ttl
		args.MinID = fmt.Sprintf("%s-0", cast.ToString(expired*1000))
	}
	return r.client.XAdd(ctx, args).Err()
}

func (r *RedisDriver) Consume(ctx context.Context, topic, group string, handler queue.ConsumeTaskHandler) error {
	err := r.initConsume(ctx, topic, group)
	if err != nil {
		return err
	}

	args := &redis.XReadGroupArgs{
		Group:    group,
		Consumer: group,
		Streams:  []string{topic, ">"},
		Block:    0,
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
				time.Sleep(100 * time.Microsecond)
				continue
			}

			err = r.client.XAck(ctx, stream.Stream, group, message.ID).Err()
			if err != nil {
				log.Printf("[redis.xAck] topic: %s, group: %s, messageId: %s, error: %v\n",
					topic, group, message.ID, err)
				continue
			}
		}
	}
}

func (r *RedisDriver) initConsume(ctx context.Context, topic, groupName string) error {
	for {
		length, err := r.client.XLen(ctx, topic).Result()
		if err != nil {
			return err
		}

		if length > 0 {
			break
		}

		// wait first message
		time.Sleep(time.Second)
		continue
	}

	groups, err := r.client.XInfoGroups(ctx, topic).Result()
	if err != nil {
		return err
	}

	for _, group := range groups {
		if groupName == group.Name {
			return nil
		}
	}
	return r.client.XGroupCreateMkStream(ctx, topic, groupName, "0").Err()
}
