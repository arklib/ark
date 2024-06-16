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
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Get(ctx context.Context, key string) (value []byte, err error)
	Del(ctx context.Context, key string) error
}

type Cache[Value any] struct {
	driver     Driver
	scene      string
	ttl        time.Duration
	serializer serializer.Serializer
}

func New[Value any](driver Driver, scene string, ttl time.Duration) Cache[Value] {
	return Cache[Value]{
		driver:     driver,
		scene:      scene,
		ttl:        ttl * time.Second,
		serializer: serializer.NewGoJson(),
	}
}

func (c Cache[Value]) WithSerializer(serialize serializer.Serializer) Cache[Value] {
	c.serializer = serialize
	return c
}

func (c Cache[Value]) Set(ctx context.Context, key any, value *Value) error {
	newKey := util.MakeStrKey(c.scene, key)
	if newKey == "" {
		return ErrKeyType
	}

	newValue, err := c.serializer.Encode(value)
	if err != nil {
		return err
	}
	return c.driver.Set(ctx, newKey, newValue, c.ttl)
}

func (c Cache[Value]) Get(ctx context.Context, key any) (value *Value, err error) {
	newKey := util.MakeStrKey(c.scene, key)
	if newKey == "" {
		err = ErrKeyType
		return
	}

	data, err := c.driver.Get(ctx, newKey)
	if err != nil {
		return
	}

	val := new(Value)
	err = c.serializer.Decode(data, val)
	return val, err
}

func (c Cache[Value]) Del(ctx context.Context, key any) error {
	newKey := util.MakeStrKey(c.scene, key)
	if newKey == "" {
		return ErrKeyType
	}
	return c.driver.Del(ctx, newKey)
}
