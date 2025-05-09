package job

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

type CornJobBuilder struct{}

func NewCornJobBuilder() *CornJobBuilder {
	return &CornJobBuilder{}
}

func (b *CornJobBuilder) Build(job Job) cron.Job {
	name := job.Name()
	fmt.Printf("Building job: %s\n", name)
	fmt.Printf("Job %s is being built\n", name)
	return cornJobFuncAdapter(func() error {
		start := time.Now()
		fmt.Printf("Job %s started at %s\n", name, start.Format(time.RFC3339))

		defer func() {
			end := time.Now()
			duration := end.Sub(start)
			fmt.Printf("Job %s finished at %s, duration: %s\n", name, end.Format(time.RFC3339), duration)
		}()
		err := job.Run()
		if err != nil {
			fmt.Printf("Job %s failed: %v\n", name, err)
		}
		return nil
	})
}

type cornJobFuncAdapter func() error

func (f cornJobFuncAdapter) Run() {
	_ = f()
}
