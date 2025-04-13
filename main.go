package main

import (
	"github.com/Fairy-nn/inspora/internal/repository"
	"github.com/Fairy-nn/inspora/internal/repository/dao"
	"github.com/Fairy-nn/inspora/internal/service"
	"github.com/Fairy-nn/inspora/internal/web"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	r := gin.Default() // 创建一个新的 Gin 路由器实例

	db := initDB()      // 初始化数据库连接
	u := initUser(db)   // 创建用户处理器
	u.Cors(r)           // 设置跨域请求头
	r = setcookie(r)    // 设置 Cookie 存储
	u.RegisterRoutes(r) // 注册路由
	r.Run(":8080")

}

func initUser(db *gorm.DB) *web.UserHandler {
	ud := dao.NewUserDAO(db)
	repo := repository.NewUserRepository(ud)
	svc := service.NewUserService(repo)
	u := web.NewUserHandler(svc) // 创建用户处理器
	return u
}

func initDB() *gorm.DB {
	db, err := gorm.Open(mysql.Open("root:root@tcp(127.0.0.1:3306)/inspora"))
	if err != nil {
		panic("连接数据库失败")
	}
	err = dao.InitDB(db) // 初始化数据库
	if err != nil {
		panic("数据库初始化失败")
	}
	return db
}

func setcookie(r *gin.Engine) *gin.Engine {
	store := cookie.NewStore([]byte("secret")) // 创建一个新的 Cookie 存储
	r.Use(sessions.Sessions("mysession", store))
	
	// 使用中间件来处理会话,登录校验
	r.Use(func(c *gin.Context) {
		if c.Request.URL.Path == "/user/login" || c.Request.URL.Path == "/user/signup" {
			c.Next() // 如果是登录或注册请求，跳过中间件
			return
		}

		session := sessions.Default(c) // 获取当前会话
		id := session.Get("userID")    // 获取会话中的用户ID
		if id == nil {
			c.AbortWithStatus(401) // 如果 ID 为空，返回 401 错误
			return
		}

		c.Set("session", session) // 将会话存储在上下文中
	})
	return r
}
