package queue

import (
	"context"
	"fmt"
	"log"

	"github.com/arklib/ark/serializer"
)

type (
	ConsumeTaskHandler func(rawMessage []byte) error
	Driver             interface {
		Produce(ctx context.Context, topic string, rawMessage []byte) error
		Consume(ctx context.Context, topic, group string, handler ConsumeTaskHandler) error
	}

	RetryPush   func(id string, rawMessage []byte) error
	RetryDriver interface {
		Init(topic, group string) error
		Add(topic, group string, rawMessage []byte, errMessage string, interval uint, failed bool) error
		Run(topic, group string, push RetryPush) error
	}

	CmdTask struct {
		Name  string
		Run   func() error
		Retry func() error
	}

	Message struct {
		Task       string `json:"task"`
		Data       any    `json:"data"`
		RetryCount uint   `json:"retryCount"`
	}

	TaskConfig struct {
		MaxRetry      uint
		RetryInterval uint
	}
	TaskHandler[Data any] func(ctx context.Context, data *Data) error
	Task[Data any]        struct {
		Name          string
		Handler       TaskHandler[Data]
		MaxRetry      uint
		RetryInterval uint
	}

	Config struct {
		Name        string
		Driver      Driver
		RetryDriver RetryDriver
		Serializer  serializer.Serializer
	}

	Queue[Data any] struct {
		Name        string
		Driver      Driver
		RetryDriver RetryDriver
		Tasks       map[string]*Task[Data]
		Serializer  serializer.Serializer
	}
)

func Define[Data any](c Config) *Queue[Data] {
	if c.Name == "" || c.Driver == nil {
		log.Fatal("[job.define] (Topic & Driver) is required.")
	}

	if c.Serializer == nil {
		c.Serializer = serializer.NewGoJson()
	}

	return &Queue[Data]{
		Name:        c.Name,
		Driver:      c.Driver,
		RetryDriver: c.RetryDriver,
		Serializer:  c.Serializer,
		Tasks:       make(map[string]*Task[Data]),
	}
}

func (q *Queue[Data]) Send(ctx context.Context, data *Data) error {
	message := &Message{
		Data: data,
	}
	rawMessage, err := q.Serializer.Encode(message)
	if err != nil {
		return err
	}
	return q.Driver.Produce(ctx, q.Name, rawMessage)
}

func (q *Queue[Data]) AddTask(name string, handler TaskHandler[Data], c TaskConfig) *Queue[Data] {
	if c.RetryInterval == 0 {
		c.RetryInterval = 15
	}

	q.Tasks[name] = &Task[Data]{
		Name:          name,
		Handler:       handler,
		MaxRetry:      c.MaxRetry,
		RetryInterval: c.RetryInterval,
	}
	return q
}

func (q *Queue[Data]) RunTask(name string) error {
	task, ok := q.Tasks[name]
	if !ok {
		err := fmt.Errorf("[queue.task] topic: %s, task: %s, undefined\n", q.Name, name)
		return err
	}

	if err := q.RetryDriver.Init(q.Name, name); err != nil {
		return err
	}

	ctx := context.Background()
	err := q.Driver.Consume(ctx, q.Name, task.Name, func(rawMessage []byte) error {
		return q.handleTask(ctx, task, rawMessage)
	})
	if err != nil {
		err = fmt.Errorf("[queue.task] topic: %s, task: %s, error: %s\n", q.Name, name, err)
		return err
	}
	return nil
}

func (q *Queue[Data]) handleTask(ctx context.Context, task *Task[Data], rawMessage []byte) error {
	data := new(Data)
	message := &Message{Data: data}

	// decode
	err := q.Serializer.Decode(rawMessage, message)
	if err != nil {
		return q.handleTaskError(task, message, err.Error())
	}

	// ignore not current task
	if message.Task != "" && message.Task != task.Name {
		return nil
	}

	// handle task
	err = task.Handler(ctx, data)
	if err != nil {
		return q.handleTaskError(task, message, err.Error())
	}
	return nil
}

func (q *Queue[Data]) handleTaskError(task *Task[Data], message *Message, errMessage string) error {
	log.Printf("[queue.task] topic: %s, task: %s, error: %s\n", q.Name, task.Name, errMessage)

	message.Task = task.Name
	message.RetryCount += 1
	isFailed := task.MaxRetry > 0 && message.RetryCount > task.MaxRetry
	if isFailed {
		message.RetryCount -= 1
	}

	rawMessage, err := q.Serializer.Encode(message)
	if err != nil {
		log.Printf("[queue.task] topic: %s, task: %s, error: %s\n", q.Name, task.Name, err)
		return err
	}

	err = q.RetryDriver.Add(q.Name, task.Name, rawMessage, errMessage, task.RetryInterval, isFailed)
	if err != nil {
		log.Printf("[retry.add] topic: %s, task: %s, error: %s\n", q.Name, task.Name, err)
		return err
	}
	return nil
}

func (q *Queue[Data]) RunRetryTask(name string) error {
	if err := q.RetryDriver.Init(q.Name, name); err != nil {
		return err
	}

	ctx := context.Background()
	push := func(id string, rawMessage []byte) error {
		log.Printf("[queue.retry] id: %s, topic: %s, task: %s\n", id, q.Name, name)
		return q.Driver.Produce(ctx, q.Name, rawMessage)
	}
	return q.RetryDriver.Run(q.Name, name, push)
}

func (q *Queue[Data]) GetCmdTasks() []*CmdTask {
	var cmdTasks []*CmdTask
	for _, task := range q.Tasks {
		cmdTask := &CmdTask{
			Name: fmt.Sprintf("%s:%s", q.Name, task.Name),
			Run: func() error {
				return q.RunTask(task.Name)
			},
			Retry: func() error {
				return q.RunRetryTask(task.Name)
			},
		}
		cmdTasks = append(cmdTasks, cmdTask)
	}
	return cmdTasks
}
