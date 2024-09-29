package task

import (
	"context"
	"encoding/json"
	"time"
)

type Driver interface {
	Pop(ctx context.Context, queue string) (data []byte, err error)
	Push(ctx context.Context, queue string, data any) error
}

type (
	Payload[Data any] struct {
		Ctx  context.Context
		Data *Data
	}

	Task[Data any] struct {
		driver  Driver
		Name    string
		Timeout time.Duration
		Handler func(Payload[Data]) error
	}
)

func New[Data any](driver Driver, name string, timeout time.Duration) Task[Data] {
	return Task[Data]{
		driver:  driver,
		Name:    name,
		Timeout: timeout,
	}
}

func (t *Task[Data]) Execute() error {
	ctx := context.Background()
	for {
		rawData, err := t.driver.Pop(ctx, t.Name)
		if err != nil {
			return err
		}

		data := new(Data)
		err = json.Unmarshal(rawData, data)
		if err != nil {
			return err
		}

		payload := Payload[Data]{Ctx: ctx, Data: data}
		err = t.Handler(payload)
		if err != nil {
			return err
		}
	}
}

func (t *Task[Data]) Push(ctx context.Context, data Data) error {
	rawData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return t.driver.Push(ctx, t.Name, rawData)
}

func (t *Task[Data]) With(handler func(Payload[Data]) error) {
	t.Handler = handler
}
