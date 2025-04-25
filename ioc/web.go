package ioc

import (
	"github.com/Fairy-nn/inspora/internal/web"
	"github.com/Fairy-nn/inspora/internal/web/middleware"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func InitGin(middlewares []gin.HandlerFunc, u *web.UserHandler) *gin.Engine {
	r := gin.Default()
	r.Use(middlewares...)
	u.RegisterRoutes(r)
	return r
}

func InitMiddlewares() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		corsMiddleware(),
		jwtMiddleware(),
		// sessionMiddleware(),
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

func jwtMiddleware() gin.HandlerFunc {
	return middleware.NewLoginMiddlewareJWT().IgnorePaths("/user/login", "/user/signup").Build()
}

func sessionMiddleware() gin.HandlerFunc {
	secret := []byte("my-secure-secret-key")
	store := cookie.NewStore(secret)
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   3600, // 设置会话过期时间（秒）
		Secure:   true, // 仅在 HTTPS 下传输
		HttpOnly: true, // 禁止 JavaScript 访问
	})
	return sessions.Sessions("mysession", store)
}
