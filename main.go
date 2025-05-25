package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Fairy-nn/inspora/ioc"
)

func main() {
	if err := InitViper(); err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	app, err := InitApp()
	if err != nil {
		panic(fmt.Errorf("failed to initialize application: %w", err))
	}

	// Initialize Elasticsearch indices
	ioc.InitializeElasticsearchIndices(app.Search)

	// 启动消费者
	for _, c := range app.Consumers {
		consumer := c // 创建一个副本以避免闭包问题
		go func() {
			err := consumer.Start(context.Background())
			if err != nil {
				panic(fmt.Errorf("failed to start consumer: %w", err))
			}
		}()
		fmt.Println("Consumer", consumer, "started in background")
	}

	// 启动定时任务
	app.Cron.Start()
	fmt.Println("Cron jobs started")

	if err := app.Server.Run(":8080"); err != nil {
		panic("failed to run server")
	}

	// Stop the cron jobs when the application is stopped
	ctx := app.Cron.Stop()
	tm := time.NewTimer(10 * time.Minute)

	// Wait for the cron jobs to finish or timeout
	select {
	case <-ctx.Done():
		fmt.Println("Cron jobs stopped")
	// If the timer cancel, select this case to continue
	case <-tm.C:
		fmt.Println("Cron jobs timed out")
	}
}
