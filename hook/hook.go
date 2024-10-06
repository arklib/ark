package hook

import (
	"context"
	"log"

	"github.com/samber/lo"
)

type (
	Next                    func() error
	Handler[Data any]       func(context.Context, *Data, Next) error
	NotifyHandler[Data any] func(context.Context, *Data) error

	Hook[Data any] struct {
		names []string
		funcs map[string]Handler[Data]

		handlers       []Handler[Data]
		notifyHandlers []NotifyHandler[Data]
	}
)

func Define[Data any](names ...string) *Hook[Data] {
	return &Hook[Data]{
		names: names,
		funcs: make(map[string]Handler[Data]),
	}
}

func (h *Hook[Data]) Notify(handlers ...NotifyHandler[Data]) *Hook[Data] {
	h.notifyHandlers = append(h.notifyHandlers, handlers...)
	return h
}

func (h *Hook[Data]) Add(name string, handler Handler[Data]) {
	if !lo.Contains(h.names, name) {
		log.Fatal("handler name is undefined")
	}
	h.funcs[name] = handler

	var handlers []Handler[Data]
	for _, n := range h.names {
		h, ok := h.funcs[n]
		if !ok {
			continue
		}
		handlers = append(handlers, h)
	}
	h.handlers = handlers
}

func (h *Hook[Data]) Emit(ctx context.Context, data *Data) error {
	var next Next
	index := 0
	next = func() error {
		if index == len(h.handlers) {
			return nil
		}
		handler := h.handlers[index]
		index++
		return handler(ctx, data, next)
	}

	if err := next(); err != nil {
		return err
	}

	// notify
	for _, handler := range h.notifyHandlers {
		if err := handler(ctx, data); err != nil {
			return err
		}
	}
	return nil
}
