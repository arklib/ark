package cache

import (
	"context"
	"errors"
	"time"

	"github.com/arklib/ark/serializer"
	"github.com/arklib/ark/util"
)

var ErrKeyType = errors.New("key type error")

type (
	Driver interface {
		Set(ctx context.Context, key string, data []byte, ttl time.Duration) error
		Get(ctx context.Context, key string) (data []byte, err error)
		Del(ctx context.Context, key string) error
	}

	Config struct {
		Driver     Driver
		Serializer serializer.Serializer
		Name       string
		TTL        uint
	}

	Cache[Data any] struct {
		driver     Driver
		serializer serializer.Serializer
		name       string
		ttl        time.Duration
	}
)

func Define[Data any](c Config) Cache[Data] {
	if c.Serializer == nil {
		c.Serializer = serializer.NewGoJson()
	}

	return Cache[Data]{
		driver:     c.Driver,
		serializer: c.Serializer,
		name:       c.Name,
		ttl:        time.Duration(c.TTL) * time.Second,
	}
}

func (c *Cache[Data]) Set(ctx context.Context, key any, data *Data) error {
	newKey := util.MakeStrKey(c.name, key)
	if newKey == "" {
		return ErrKeyType
	}

	newData, err := c.serializer.Encode(data)
	if err != nil {
		return err
	}
	return c.driver.Set(ctx, newKey, newData, c.ttl)
}

func (c *Cache[Data]) Get(ctx context.Context, key any) (*Data, error) {
	newKey := util.MakeStrKey(c.name, key)
	if newKey == "" {
		return nil, ErrKeyType
	}

	rawData, err := c.driver.Get(ctx, newKey)
	if err != nil {
		return nil, err
	}

	data := new(Data)
	err = c.serializer.Decode(rawData, data)
	return data, err
}

func (c *Cache[Data]) GetOrSet(ctx context.Context, key any, handler func() (*Data, error)) (*Data, error) {
	data, err := c.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	data, err = handler()
	if err != nil {
		return nil, err
	}

	err = c.Set(ctx, key, data)
	return data, err
}

func (c *Cache[Data]) Del(ctx context.Context, key any) error {
	newKey := util.MakeStrKey(c.name, key)
	if newKey == "" {
		return ErrKeyType
	}
	return c.driver.Del(ctx, newKey)
}
