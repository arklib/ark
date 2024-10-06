package queue

import (
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"

	"github.com/samber/lo"
)

type CmdQueue interface {
	GetCmdTasks() []*CmdTask
}

func GetTasks(queues any, names ...string) []*CmdTask {
	if len(names) > 0 && names[0] == "all" {
		names = names[1:]
	}

	var tasks []*CmdTask
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
			case len(names) == 0, lo.Contains(names, cmdTask.Name):
				tasks = append(tasks, cmdTask)
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
	for _, task := range GetTasks(queues) {
		fmt.Printf("* %s\n", task.Name)
	}
}

func Run(queues any, names []string, concurrent int) {
	if len(names) == 0 {
		PrintList(queues)
		return
	}

	if concurrent <= 0 {
		concurrent = 1
	}

	var wg sync.WaitGroup
	tasks := GetTasks(queues, names...)
	taskCh := make(chan *CmdTask, len(tasks)*concurrent)

	for _, task := range tasks {
		for i := 0; i < concurrent; i++ {
			number := i + 1
			wg.Add(1)
			go func(t *CmdTask) {
				defer wg.Done()
				for {
					taskCh <- t
					err := t.Run()
					if err != nil {
						log.Printf("[task.run] name: %s, number: %d, error: %s\n", t.Name, number, err)
					}
					time.Sleep(time.Second)
					<-taskCh
				}
			}(task)
		}
		log.Printf("[task.run] name: %s, concurrent: %d", task.Name, concurrent)
	}
	wg.Wait()
}

func RunRetry(queues any, names []string) {
	if len(names) == 0 {
		PrintList(queues)
		return
	}

	taskList := GetTasks(queues, names...)
	for {
		for _, task := range taskList {
			if err := task.Retry(); err != nil {
				log.Printf("[task.retry] topic: %s, error: %s\n", task.Name, err.Error())
				continue
			}
		}
		time.Sleep(time.Second)
	}
}
