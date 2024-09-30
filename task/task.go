package task

import (
	"fmt"
	"log"
	"time"
)

type Task struct {
	list map[string]func() error
}

func New() *Task {
	return &Task{
		map[string]func() error{},
	}
}

func (t *Task) Register(name string, handler func() error) {
	t.list[name] = handler
}

func (t *Task) GetList() []string {
	var names []string
	for k := range t.list {
		names = append(names, k)
	}
	return names
}

func (t *Task) PrintList() {
	fmt.Println("tasks:")
	for _, name := range t.GetList() {
		fmt.Printf("* %s\n", name)
	}
}

func (t *Task) Run(args []string) {
	if len(args) == 0 {
		t.PrintList()
		return
	}

	name := args[0]
	handler, ok := t.list[name]
	if !ok {
		log.Printf("[task] '%s', error: 'not found'\n", name)
		return
	}

	start := time.Now()
	err := handler()
	elapsed := time.Since(start)
	if err != nil {
		log.Printf("[task] '%s', elapsed: %v, error: '%s'\n", name, elapsed, err)
		return
	}
	log.Printf("[task] '%s', elapsed: %v", name, elapsed)
}
