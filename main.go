package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	if err := InitViper(); err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	// r := InitInspora()
	// r.Run(":8080") // 启动服务器
	app := InitApp() // 初始化应用程序

	// 启动消费者
	for _, c := range app.Consumers {
		err := c.Start(context.Background())
		if err != nil {
			panic(fmt.Errorf("failed to start consumer: %w", err))
		}
		fmt.Println("Consumer ", c, "started")
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
