package main

import (
	"context"
	"fmt"
)

func main() {
	if err := InitViper(); err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	// r := InitInspora()
	// r.Run(":8080") // 启动服务器
	app := InitApp() // 初始化应用程序

	for _, c := range app.Consumers {
		err := c.Start(context.Background())
		if err != nil {
			panic(fmt.Errorf("failed to start consumer: %w", err))
		}
		fmt.Println("Consumer ", c, "started")
	}

	if err := app.Server.Run(":8080"); err != nil {
		panic("failed to run server")
	}
}
