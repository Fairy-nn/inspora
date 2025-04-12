package web

import (
	"regexp"

	"github.com/gin-gonic/gin"
)

// 用户有关的路由
type UserHandler struct {
}

// 注册
func (u *UserHandler) SignUp(ctx *gin.Context) {
	const (
		emailRegex    = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
		passwordRegex = `^[a-zA-Z0-9]{6,16}$` //仅包含字母和数字，长度在 6 - 16 位
	)
	type SignUpReq struct {
		Email           string `json:"email"`
		Username        string `json:"username"`
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirm_password"`
	}
	var req SignUpReq
	if err := ctx.Bind(&req); err != nil {
		ctx.JSON(400, gin.H{"error": "Invalid request"})
		return
	}
	// 正则表达式验证邮箱格式
	_, error := regexp.Match(emailRegex, []byte(req.Email))
	if error != nil {
		// 记录错误日志
		ctx.JSON(400, gin.H{"error": "邮件格式不正确"})
		return
	}
	// 正则表达式验证密码格式
	_, error = regexp.Match(passwordRegex, []byte(req.Password))
	if error != nil {
		ctx.JSON(400, gin.H{"error": "密码格式不正确，仅包含字母和数字，长度在 6-16 位"})
		return
	}
	// 验证密码和确认密码是否一致
	if req.Password != req.ConfirmPassword {
		ctx.JSON(400, gin.H{"error": "两次密码不一致"})
		return
	}

	ctx.JSON(200, gin.H{"message": "注册成功"})
}

// 登录
func (u *UserHandler) Login(ctx *gin.Context) {

}

// 编辑用户信息
func (u *UserHandler) Edit(ctx *gin.Context) {

}

// 获取用户信息
func (u *UserHandler) Profile(ctx *gin.Context) {

}
