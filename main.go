package main

import "fmt"

func main() {
	if err := InitViper(); err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	r := InitInspora()
	r.Run(":8080") // 启动服务器
}
