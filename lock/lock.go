package lock

import (
	"context"
	"errors"
	"time"

	"github.com/arklib/ark/util"
)

var ErrKeyType = errors.New("key type error")
var ErrIsLocked = errors.New("is locked")

type (
	Driver interface {
		Lock(ctx context.Context, key string, ttl time.Duration) (bool, error)
		Unlock(ctx context.Context, key string) error
	}

	Config struct {
		Driver Driver
		Name   string
		TTL    uint
	}

	Payload struct {
		driver Driver
		key    string
		ctx    context.Context
	}

	Lock struct {
		driver Driver
		name   string
		ttl    time.Duration
	}
)

func Define(c Config) *Lock {
	return &Lock{
		driver: c.Driver,
		name:   c.Name,
		ttl:    time.Duration(c.TTL) * time.Second,
	}
}

func (l *Lock) Lock(ctx context.Context, key any) (payload *Payload, err error) {
	strKey := util.MakeStrKey(l.name, key)
	if strKey == "" {
		err = ErrKeyType
		return
	}

	lock, err := l.driver.Lock(ctx, strKey, l.ttl)
	if err != nil {
		return
	}
	if !lock {
		err = ErrIsLocked
		return
	}

	payload = &Payload{
		ctx:    ctx,
		driver: l.driver,
		key:    strKey,
	}
	return
}

func (p *Payload) Unlock() error {
	return p.driver.Unlock(p.ctx, p.key)
}
