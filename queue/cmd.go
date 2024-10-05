package queue

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/samber/lo"
)

type CmdQueue interface {
	GetCmdTasks() []*CmdTask
}

func GetTaskList(queues any, filter ...string) map[string]*CmdTask {
	if len(filter) > 0 && filter[0] == "all" {
		filter = filter[1:]
	}
	tasks := make(map[string]*CmdTask)

	rQueues := reflect.ValueOf(queues).Elem()
	for i := 0; i < rQueues.NumField(); i++ {
		rQueue := rQueues.Field(i)
		if rQueue.Elem().Kind() != reflect.Struct {
			continue
		}

		queue := rQueue.Interface().(CmdQueue)
		if queue == nil {
			continue
		}

		for _, cmdTask := range queue.GetCmdTasks() {
			switch {
			case len(filter) == 0, lo.Contains(filter, cmdTask.Name):
				tasks[cmdTask.Name] = cmdTask
			default:
				continue
			}
		}
	}
	return tasks
}

func PrintList(queues any) {
	fmt.Println("tasks:")
	fmt.Println("* all")
	for name, _ := range GetTaskList(queues) {
		fmt.Printf("* %s\n", name)
	}
}

func Run(queues any, tasks []string) {
	if len(tasks) == 0 {
		PrintList(queues)
		return
	}

	for _, task := range GetTaskList(queues, tasks...) {
		go func() {
			if err := task.Run(); err != nil {
				log.Print(err)
			}
		}()
	}
	select {}
}

func RunRetry(queues any, tasks []string) {
	if len(tasks) == 0 {
		PrintList(queues)
		return
	}

	taskList := GetTaskList(queues, tasks...)
	for {
		for _, task := range taskList {
			if err := task.Retry(); err != nil {
				log.Printf("[task.retry] topic: %s, error: %s\n", task.Name, err.Error())
				continue
			}
		}
		time.Sleep(time.Second * 1)
	}
}
