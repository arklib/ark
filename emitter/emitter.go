package emitter

import (
	"context"
)

type (
	Payload[Data any] struct {
		Ctx  context.Context
		Data *Data
		Next func() error
	}
	Emitter[Data any] func(context.Context, *Data) error
)

func New[Data any](handlers ...func(Payload[Data]) error) Emitter[Data] {
	return func(ctx context.Context, data *Data) error {
		p := Payload[Data]{
			Ctx:  ctx,
			Data: data,
		}

		index := 0
		p.Next = func() error {
			if index == len(handlers) {
				return nil
			}
			handler := handlers[index]
			index++
			return handler(p)
		}
		return p.Next()
	}
}
