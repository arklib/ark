package job

import (
	"context"
	"log"
	"time"

	"github.com/arklib/ark/serializer"
)

type (
	PushCallback func(id string, data []byte) error

	Driver interface {
		Pop(ctx context.Context, queue string) (rawData []byte, err error)
		Push(ctx context.Context, queue string, rawData []byte) error
		// Ack(ctx context.Context, queue string, data any) error
	}

	RetryDriver interface {
		Init() error
		Add(queue string, rawData []byte, retryTime uint, errMsg string) error
		Run(queue string, push PushCallback) error
	}

	Config struct {
		Driver      Driver
		RetryDriver RetryDriver
		Serializer  serializer.Serializer

		Queue     string
		Timeout   uint
		RetryTime uint
	}

	Payload[Data any] struct {
		Ctx  context.Context
		Data *Data
	}

	Job[Data any] struct {
		driver      Driver
		retryDriver RetryDriver
		serializer  serializer.Serializer

		queue     string
		timeout   uint
		retryTime uint
		handlers  []func(Payload[Data]) error
	}

	Cmd struct {
		Name  string
		Run   func() error
		Retry func() error
	}
)

func Define[Data any](c Config) *Job[Data] {
	if c.Serializer == nil {
		c.Serializer = serializer.NewGoJson()
	}

	if c.RetryTime == 0 {
		c.RetryTime = 30
	}

	return &Job[Data]{
		driver:      c.Driver,
		retryDriver: c.RetryDriver,
		serializer:  c.Serializer,
		queue:       c.Queue,
		timeout:     c.Timeout,
		retryTime:   c.RetryTime,
	}
}

func (t *Job[Data]) GetCmd() *Cmd {
	return &Cmd{
		Name:  t.queue,
		Run:   t.Run,
		Retry: t.Retry,
	}
}

func (t *Job[Data]) Use(handler func(Payload[Data]) error) {
	t.handlers = append(t.handlers, handler)
}

func (t *Job[Data]) Dispatch(ctx context.Context, data Data) error {
	rawData, err := t.serializer.Encode(data)
	if err != nil {
		return err
	}
	return t.driver.Push(ctx, t.queue, rawData)
}

func (t *Job[Data]) Run() error {
	if err := t.retryDriver.Init(); err != nil {
		return err
	}

	ctx := context.Background()
	for {
		rawData, err := t.driver.Pop(ctx, t.queue)
		if err != nil {
			log.Printf("[job] name: '%s', error: '%s'", t.queue, err)
			time.Sleep(time.Second)
			continue
		}

		if len(rawData) == 0 {
			time.Sleep(time.Second)
			continue
		}

		data := new(Data)
		err = t.serializer.Decode(rawData, data)
		if err != nil {
			_ = t.retryDriver.Add(t.queue, rawData, t.retryTime, err.Error())
			continue
		}

		payload := Payload[Data]{Ctx: ctx, Data: data}
		for _, handler := range t.handlers {
			err = handler(payload)
			if err != nil {
				_ = t.retryDriver.Add(t.queue, rawData, t.retryTime, err.Error())
				break
			}
		}
	}
}

func (t *Job[Data]) Retry() error {
	if err := t.retryDriver.Init(); err != nil {
		return err
	}

	ctx := context.Background()
	push := func(id string, data []byte) error {
		log.Printf("[job.retry] jobId: %s, name: %s\n", id, t.queue)
		return t.driver.Push(ctx, t.queue, data)
	}
	return t.retryDriver.Run(t.queue, push)
}
