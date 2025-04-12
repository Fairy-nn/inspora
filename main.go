package main

import (
	"github.com/Fairy-nn/inspora/internal/web"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default() // 创建一个新的 Gin 路由器实例

	web.Cors(r) // CORS 跨域设置

	web.UserInitRouts(r) // 初始化路由

	r.Run(":8080") // 启动服务器，监听在 8080 端口

}
