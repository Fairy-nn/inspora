package middleware

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

type LoginMiddlewareJWT struct {
	paths []string
}

func NewLoginMiddlewareJWT() *LoginMiddlewareJWT {
	return &LoginMiddlewareJWT{}
}

// 用于设置白名单路径的函数
func (b *LoginMiddlewareJWT) IgnorePaths(paths ...string) *LoginMiddlewareJWT {
	b.paths = paths
	return b
}

// loginMiddleware 中间件函数
func (b *LoginMiddlewareJWT) Build() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取请求的路径
		path := c.Request.URL.Path

		// 检查路径是否在白名单中
		for _, p := range b.paths {
			if strings.HasPrefix(path, p) { // 检查路径是否以白名单路径开头
				return
			}
		}

		// 获取请求头中的Authorization字段
		token := c.Request.Header.Get("Authorization")
		if token == "" {
			c.JSON(401, gin.H{"error": "未登录或无效的token"})
			return
		}

		// 分割Authorization字段，获取token
		segs := strings.Split(token, " ")
		if len(segs) != 2 || segs[0] != "Bearer" {
			c.JSON(401, gin.H{"error": "未登录或无效的token"})
			return
		}

		// 解析token
		tokenStr := segs[1]
		parsedToken, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			// 验证签名方法是否正确
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.NewValidationError("签名方法错误", jwt.ValidationErrorSignatureInvalid)
			}
			// 返回密钥
			return []byte("my_secret_key"), nil
		})

		// 检查token是否有效
		if err != nil || !parsedToken.Valid {
			c.JSON(401, gin.H{"error": "未登录或无效的token"})
			return
		}

		// 将解析后的token中的claims存入上下文中
		if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok {
			fmt.Printf("%+v\n", claims) // DEBUG: 打印claims信息

			c.Set("claims", claims) // 将用户ID存入上下文中
		}
	}
}
