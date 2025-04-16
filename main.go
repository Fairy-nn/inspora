package main

import (
	"github.com/Fairy-nn/inspora/internal/repository"
	"github.com/Fairy-nn/inspora/internal/repository/dao"
	"github.com/Fairy-nn/inspora/internal/service"
	"github.com/Fairy-nn/inspora/internal/web"
	"github.com/Fairy-nn/inspora/internal/web/middleware"
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
	// 从环境变量加载密钥
	secret := []byte("my-secure-secret-key")
	store := cookie.NewStore(secret)
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   3600, // 设置会话过期时间（秒）
		Secure:   true, // 仅在 HTTPS 下传输
		HttpOnly: true, // 禁止 JavaScript 访问
	})
	r.Use(sessions.Sessions("mysession", store)) // 使用 Cookie 存储会话
	//r.Use(middleware.NewLoginMiddleware().IgnorePaths("/users/login", "/users/signup").Build()) // 使用自定义的会话中间件
	r.Use(middleware.NewLoginMiddlewareJWT().IgnorePaths("/user/login", "/user/signup").Build()) // 使用 JWT 中间件
	return r
}
