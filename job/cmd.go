package job

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/samber/lo"
)

type CmdJob interface {
	GetCmd() *Cmd
}

type CmdJobs = map[string]*Cmd

func GetListByFilter(jobs any, filter []string) CmdJobs {
	cmdJobs := make(CmdJobs)

	rJobs := reflect.ValueOf(jobs).Elem()
	for i := 0; i < rJobs.NumField(); i++ {
		rJob := rJobs.Field(i)
		if rJob.Elem().Kind() != reflect.Struct {
			continue
		}

		job := rJob.Interface().(CmdJob)
		if job == nil {
			continue
		}

		cmd := job.GetCmd()
		if len(filter) > 0 && filter[0] != "all" && !lo.Contains(filter, cmd.Queue) {
			continue
		}
		cmdJobs[cmd.Queue] = cmd
	}
	return cmdJobs
}

func GetList(jobs any) CmdJobs {
	return GetListByFilter(jobs, []string{})
}

func PrintList(jobs any) {
	fmt.Println("jobs:")
	fmt.Println("* all")
	for name, _ := range GetList(jobs) {
		fmt.Printf("* %s\n", name)
	}
}

func Run(jobs any, queues []string) {
	if len(queues) == 0 {
		PrintList(jobs)
		return
	}

	for _, job := range GetListByFilter(jobs, queues) {
		go func() {
			if err := job.Run(); err != nil {
				log.Print(err)
			}
		}()
	}
	select {}
}

func RunRetry(jobs any, queues []string) {
	if len(queues) == 0 {
		PrintList(jobs)
		return
	}

	jobList := GetListByFilter(jobs, queues)
	for {
		for _, job := range jobList {
			if err := job.Retry(); err != nil {
				log.Printf("[job.retry] queue: %s, error: %s\n", job.Queue, err.Error())
				continue
			}
		}
		time.Sleep(time.Second * 1)
	}
}
