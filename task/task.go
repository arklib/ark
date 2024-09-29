package task

import (
	"context"
	"time"

	"github.com/arklib/ark/serializer"
)

type (
	Driver interface {
		Pop(ctx context.Context, queue string) (data []byte, err error)
		Push(ctx context.Context, queue string, data any) error
	}

	Config struct {
		Driver     Driver
		Serializer serializer.Serializer
		Queue      string
		Timeout    uint
	}

	Payload[Data any] struct {
		Ctx  context.Context
		Data *Data
	}

	Task[Data any] struct {
		driver     Driver
		serializer serializer.Serializer
		queue      string
		timeout    time.Duration
		handler    func(Payload[Data]) error
	}
)

func Define[Data any](c Config) Task[Data] {
	if c.Serializer == nil {
		c.Serializer = serializer.NewGoJson()
	}

	return Task[Data]{
		driver:     c.Driver,
		serializer: c.Serializer,
		queue:      c.Queue,
		timeout:    time.Duration(c.Timeout) * time.Second,
	}
}

func (t *Task[Data]) Execute() error {
	ctx := context.Background()
	for {
		rawData, err := t.driver.Pop(ctx, t.queue)
		if err != nil {
			return err
		}

		data := new(Data)
		err = t.serializer.Decode(rawData, data)
		if err != nil {
			return err
		}

		payload := Payload[Data]{Ctx: ctx, Data: data}
		err = t.handler(payload)
		if err != nil {
			return err
		}
	}
}

func (t *Task[Data]) Push(ctx context.Context, data Data) error {
	rawData, err := t.serializer.Encode(data)
	if err != nil {
		return err
	}
	return t.driver.Push(ctx, t.queue, rawData)
}

func (t *Task[Data]) With(handler func(Payload[Data]) error) {
	t.handler = handler
}
