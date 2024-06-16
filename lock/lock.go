package lock

import (
	"context"
	"errors"
	"time"

	"github.com/arklib/ark/util"
)

var ErrKeyType = errors.New("key type error")
var ErrIsLocked = errors.New("is locked")

type Driver interface {
	Set(ctx context.Context, key string, value any, ttl time.Duration) (bool, error)
	Del(ctx context.Context, key string) error
}

type Locked struct {
	driver Driver
	key    string
	ctx    context.Context
}

type Lock struct {
	driver Driver
	scene  string
	ttl    time.Duration
}

func New(driver Driver, scene string, ttl time.Duration) *Lock {
	return &Lock{
		driver: driver,
		scene:  scene,
		ttl:    ttl,
	}
}

func (l *Lock) Apply(ctx context.Context, key any) (locked *Locked, err error) {
	newKey := util.MakeStrKey(l.scene, key)
	if newKey == "" {
		err = ErrKeyType
		return
	}

	lock, err := l.driver.Set(ctx, newKey, 1, l.ttl)
	if err != nil {
		return
	}
	if !lock {
		err = ErrIsLocked
		return
	}
	locked = &Locked{
		ctx:    ctx,
		driver: l.driver,
		key:    newKey,
	}
	return
}

func (l *Locked) Release() error {
	return l.driver.Del(l.ctx, l.key)
}
