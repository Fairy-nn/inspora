package web

import "github.com/gin-gonic/gin"

// UserInitRouts 初始化路由
func UserInitRouts(r *gin.Engine) {
	var u *UserHandler
	ug := r.Group("/user") // 用户相关路由
	ug.POST("/signup", u.SignUp)
	ug.POST("/login", u.Login)
	ug.PUT("/edit", u.Edit)
	ug.GET("/profile", u.Profile)
}

// Cors 设置
func Cors(r *gin.Engine) {
	// 设置 CORS 跨域请求头
	r.Use(func(c *gin.Context) {
		// 允许的域名
        allowedOrigin := "https://localhost:8080"
        c.Header("Access-Control-Allow-Origin", allowedOrigin)          
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")     // 允许的请求头
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS") // 允许的请求方法
		c.Header("Access-Control-Allow-Credentials", "true")                        // 允许携带凭证
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
	})
}
