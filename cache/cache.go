package cache

import (
	"context"
	"errors"
	"time"

	"github.com/arklib/ark/serializer"
	"github.com/arklib/ark/util"
)

var ErrKeyType = errors.New("key type error")

type Driver interface {
	Set(ctx context.Context, key string, data any, ttl time.Duration) error
	Get(ctx context.Context, key string) (data []byte, err error)
	Del(ctx context.Context, key string) error
}

type Cache[Data any] struct {
	driver     Driver
	scene      string
	ttl        time.Duration
	serializer serializer.Serializer
}

func New[Data any](driver Driver, scene string, ttl time.Duration) Cache[Data] {
	return Cache[Data]{
		driver:     driver,
		scene:      scene,
		ttl:        ttl * time.Second,
		serializer: serializer.NewGoJson(),
	}
}

func (c Cache[Data]) WithSerializer(serialize serializer.Serializer) Cache[Data] {
	c.serializer = serialize
	return c
}

func (c Cache[Data]) Set(ctx context.Context, key any, data *Data) error {
	newKey := util.MakeStrKey(c.scene, key)
	if newKey == "" {
		return ErrKeyType
	}

	newData, err := c.serializer.Encode(data)
	if err != nil {
		return err
	}
	return c.driver.Set(ctx, newKey, newData, c.ttl)
}

func (c Cache[Data]) Get(ctx context.Context, key any) (data *Data, err error) {
	newKey := util.MakeStrKey(c.scene, key)
	if newKey == "" {
		err = ErrKeyType
		return
	}

	rawData, err := c.driver.Get(ctx, newKey)
	if err != nil {
		return
	}

	data = new(Data)
	err = c.serializer.Decode(rawData, data)
	return data, err
}

func (c Cache[Data]) Del(ctx context.Context, key any) error {
	newKey := util.MakeStrKey(c.scene, key)
	if newKey == "" {
		return ErrKeyType
	}
	return c.driver.Del(ctx, newKey)
}
