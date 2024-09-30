package event

import (
	"context"
)

type (
	Payload[Data any] struct {
		Ctx  context.Context
		Data *Data
		Next func() error
	}

	Event[Data any] struct {
		handlers []func(Payload[Data]) error
	}
)

func Define[Data any]() *Event[Data] {
	return &Event[Data]{}
}

func (e Event[Data]) Use(handler ...func(Payload[Data]) error) {
	e.handlers = append(e.handlers, handler...)
}

func (e Event[Data]) Dispatch(ctx context.Context, data *Data) error {
	p := Payload[Data]{
		Ctx:  ctx,
		Data: data,
	}

	index := 0
	p.Next = func() error {
		if index == len(e.handlers) {
			return nil
		}
		handler := e.handlers[index]
		index++
		return handler(p)
	}
	return p.Next()
}
