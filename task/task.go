package task

import (
	"fmt"
	"log"
)

type Handler = func() error
type Task struct {
	handlers map[string]Handler
}

func New() *Task {
	return &Task{
		handlers: make(map[string]Handler),
	}
}

func (t *Task) PrintList() {
	fmt.Println("tasks:")
	for name, _ := range t.handlers {
		fmt.Printf("* %s\n", name)
	}
}

func (t *Task) Add(name string, handler Handler) {
	t.handlers[name] = handler
}

func (t *Task) Run(names ...string) {
	if len(names) == 0 {
		t.PrintList()
		return
	}

	for _, name := range names {
		if handler, ok := t.handlers[name]; ok {
			if err := handler(); err != nil {
				log.Printf("[%s] error: %s\n", name, err)
				continue
			}
			log.Printf("[%s] done\n", name)
		}
	}
}
