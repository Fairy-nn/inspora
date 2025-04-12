package main

import (
	"github.com/gin-gonic/gin"
	"github.com/Fairy-nn/inspora/internal/web"
)

func main() {
	// 创建一个新的 Gin 路由器实例
	r := gin.Default()
	// 初始化路由
	web.UserInitRouts(r)
	// 启动服务器，监听在 8080 端口
	r.Run(":8080")

}