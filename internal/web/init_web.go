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