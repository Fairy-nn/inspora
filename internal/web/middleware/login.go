package middleware

import (
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type LoginMiddleware struct {
	paths []string
}

func NewLoginMiddleware() *LoginMiddleware {
	return &LoginMiddleware{}
}

// 用于设置白名单路径的函数
func (b *LoginMiddleware) IgnorePaths(paths ...string) *LoginMiddleware {
	b.paths = paths
	return b
}

// loginMiddleware 中间件函数
// 该函数用于检查用户是否已登录，如果未登录，则返回 401 错误
// 同时还会检查会话的更新时间，如果超过 1 小时，则更新会话
func (b *LoginMiddleware) Build() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取请求的路径
		path := c.Request.URL.Path

		// 检查路径是否在白名单中
		for _, p := range b.paths {
			if strings.HasPrefix(path, p) { // 检查路径是否以白名单路径开头
				c.Next()
				return
			}
		}

		session := sessions.Default(c) // 获取当前会话
		id := session.Get("userID")    // 获取会话中的用户ID

		if id == nil { // 还没有登录，ID 为空
			c.AbortWithStatus(401) // 如果 ID 为空，返回 401 错误
			return
		}

		c.Set("userID", id)

		updateTime := session.Get("updateTime") // 获取会话中的更新时间

		if updateTime == nil { // 第一次登录，还没有更新时间
			session.Set("updateTime", time.Now().UnixNano()/int64(time.Millisecond)) // 存储为毫秒时间戳
			session.Save()
			return
		}

		updateTimeVal, ok := updateTime.(int64) // 确保类型为 int64
		if !ok {
			c.AbortWithStatus(500) // 如果类型不匹配，返回服务器错误
			return
		}

		now := time.Now().UnixNano() / int64(time.Millisecond) // 获取当前时间（毫秒）
		if now-updateTimeVal > 3600*1000 {                     // 如果超过 1 小时，更新会话
			session.Set("updateTime", now) // 更新会话中的更新时间
			session.Save()
		}

		c.Next() // 继续处理请求
	}
}
