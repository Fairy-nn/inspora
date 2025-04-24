package web

import (
	"fmt"
	"regexp"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/service"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

// 用户有关的路由
type UserHandler struct {
	svc         *service.UserService // 用户服务
	emailExp    *regexp.Regexp       // 邮箱正则表达式
	passwordExp *regexp.Regexp       // 密码正则表达式
	codeSvc     *service.CodeService // 短信验证码服务
}

// RegisterRoutes 注册路由
func (u *UserHandler) RegisterRoutes(r *gin.Engine) {
	ug := r.Group("/user")                  // 用户相关路由
	ug.POST("/signup", u.SignUp)            // 注册
	ug.POST("/login", u.LoginJWT)           // 登录
	ug.PUT("/edit", u.Edit)                 // 编辑用户信息
	ug.GET("/profile", u.Profile)           // 获取用户信息
	ug.POST("/login_sms/send", u.SendSMS)   // 发送短信验证码
	ug.POST("/login_sms/login", u.LoginSMS) // 验证短信验证码
}

// Cors 设置
func (u *UserHandler) Cors(r *gin.Engine) {
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

// NewUserHandler 创建用户处理器
// 该函数用于创建一个新的用户处理器实例，接收一个用户服务作为参数
func NewUserHandler(svc *service.UserService) *UserHandler {
	const (
		emailRegex    = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
		passwordRegex = `^[a-zA-Z0-9]{6,16}$` //仅包含字母和数字，长度在 6 - 16 位
	)

	emailExp := regexp.MustCompile(emailRegex)
	passwordExp := regexp.MustCompile(passwordRegex)

	return &UserHandler{
		svc:         svc,
		emailExp:    emailExp,
		passwordExp: passwordExp,
	}
}

// 注册
func (u *UserHandler) SignUp(ctx *gin.Context) {
	// 定义请求体结构体
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
	_, error := regexp.Match(u.emailExp.String(), []byte(req.Email))
	if error != nil {
		// 记录错误日志
		ctx.JSON(400, gin.H{"error": "邮件格式不正确"})
		return
	}
	// 正则表达式验证密码格式
	_, error = regexp.Match(u.passwordExp.String(), []byte(req.Password))
	if error != nil {
		ctx.JSON(400, gin.H{"error": "密码格式不正确，仅包含字母和数字，长度在 6-16 位"})
		return
	}
	// 验证密码和确认密码是否一致
	if req.Password != req.ConfirmPassword {
		ctx.JSON(400, gin.H{"error": "两次密码不一致"})
		return
	}
	// 调用服务层的注册方法
	err := u.svc.SignUp(ctx, domain.User{
		Email:    req.Email,
		Password: req.Password,
		Username: req.Username,
	})

	if err != nil {
		if err.Error() == "用户邮箱已存在" {
			ctx.JSON(400, gin.H{"error": "用户邮箱已存在"})
			return
		}
		ctx.JSON(500, gin.H{"error": "注册失败"})
		return
	}

	ctx.JSON(200, gin.H{"message": "注册成功"})
}

// 登录
func (u *UserHandler) Login(ctx *gin.Context) {
	// 定义请求体结构体
	type LoginReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var req LoginReq
	if err := ctx.Bind(&req); err != nil { // 绑定请求体到结构体
		ctx.JSON(400, gin.H{"error": "请求体格式错误"})
		return
	}

	user, err := u.svc.Login(ctx, domain.User{
		Email:    req.Email,
		Password: req.Password,
	}) // 调用服务层的登录方法

	if err != nil {
		if err.Error() == "密码或邮箱不正确" {
			ctx.JSON(400, gin.H{"error": "密码或邮箱不正确"})
			return
		} else if err.Error() == "用户不存在" {
			ctx.JSON(400, gin.H{"error": "用户不存在"})
			return
		}
		ctx.JSON(500, gin.H{"error": "登录失败"})
		return
	}
	// 设置session
	session := sessions.Default(ctx) // 获取session
	session.Set("userID", user.ID)   // 将用户ID存入session
	session.Save()

	ctx.JSON(200, gin.H{"message": "登录成功"}) // 返回登录成功的响应
}

// 登录使用JWT
func (u *UserHandler) LoginJWT(ctx *gin.Context) {
	// 定义请求体结构体
	type LoginReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var req LoginReq
	if err := ctx.Bind(&req); err != nil { // 绑定请求体到结构体
		ctx.JSON(400, gin.H{"error": "请求体格式错误"})
		return
	}

	user, err := u.svc.Login(ctx, domain.User{
		Email:    req.Email,
		Password: req.Password,
	}) // 调用服务层的登录方法
	fmt.Print("这是用户ID:", user.ID)
	fmt.Println("这是用户emile:", user.Email)

	if err != nil {
		if err.Error() == "密码或邮箱不正确" {
			ctx.JSON(400, gin.H{"error": "密码或邮箱不正确"})
			return
		} else if err.Error() == "用户不存在" {
			ctx.JSON(400, gin.H{"error": "用户不存在"})
			return
		}
		ctx.JSON(500, gin.H{"error": "登录失败"})
		return
	}

	// 生成JWT令牌
	claims := jwt.MapClaims{
		"id":  user.ID,                                   // 用户ID
		"exp": jwt.TimeFunc().Add(time.Hour * 24).Unix(), // 过期时间为24小时
		"iat": jwt.TimeFunc().Unix(),                     // 签发时间
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("my_secret_key"))
	if err != nil {
		ctx.JSON(500, gin.H{"error": "生成JWT失败"})
		return
	}
	ctx.Header("jwt", tokenString) // 将JWT令牌添加到响应头中

	ctx.JSON(200, gin.H{"message": "登录成功"}) // 返回登录成功的响应
}

// 获取用户信息
func (u *UserHandler) Profile(ctx *gin.Context) {
	// 类型断言
	c, _ := ctx.Get("claims")
	claims, ok := c.(jwt.MapClaims) // 将 claims 转换为 jwt.MapClaims 类型，以便访问其中的字段
	if !ok {
		ctx.JSON(400, gin.H{"error": "获取用户信息失败"})
		return
	}

	fmt.Println("这是用户ID:", claims["id"])
}

// 编辑用户信息
func (u *UserHandler) Edit(ctx *gin.Context) {

}

// 发送验证码并验证手机号码是否符合格式
func (u *UserHandler) SendSMS(ctx *gin.Context) {
	type SendSMSRequest struct {
		Phone string `json:"phone"`
	}
	var req SendSMSRequest
	if err := ctx.Bind(&req); err != nil {
		ctx.JSON(400, gin.H{"error": "请求体格式错误"})
		return
	}
	// 正则表达式验证手机号格式
	phoneExp := regexp.MustCompile(`^1[3-9]\d{9}$`)
	if !phoneExp.MatchString(req.Phone) {
		ctx.JSON(400, gin.H{"error": "手机号格式不正确"})
		return
	}
	// 调用服务层的发送验证码方法
	err := u.codeSvc.Send(ctx, "login", req.Phone) // 发送验证码
	if err != nil {
		if err.Error() == "验证码存储失败" {
			ctx.JSON(500, gin.H{"error": "系统异常，请稍后再试"})
			return
		} else if err.Error() == "验证码发送失败" {
			ctx.JSON(500, gin.H{"error": "验证码发送失败"})
			return
		}
		ctx.JSON(500, gin.H{"error": "发送验证码失败"})
		return
	}
	ctx.JSON(200, gin.H{"message": "验证码发送成功"})
}

// 校验短信验证码并使用验证码登录
func (u *UserHandler) LoginSMS(ctx *gin.Context) {
	type LoginRequest struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	var req LoginRequest
	if err := ctx.Bind(&req); err != nil {
		ctx.JSON(400, gin.H{"error": "请求体格式错误"})
		return
	}
	// 正则表达式验证手机号格式
	phoneExp := regexp.MustCompile(`^1[3-9]\d{9}$`)
	if !phoneExp.MatchString(req.Phone) {
		ctx.JSON(400, gin.H{"error": "手机号格式不正确"})
		return
	}
	// 校验验证码
	_, err := u.codeSvc.Verify(ctx, "login", req.Phone, req.Code) // 校验验证码
	if err != nil {
		if err.Error() == "验证码验证失败" {
			ctx.JSON(500, gin.H{"error": "验证码验证失败"})
			return
		}
		ctx.JSON(500, gin.H{"error": "系统异常，请稍后再试"})
		return
	}

	// 登录成功，获取用户信息,手机号码可能是新用户，所以需要根据手机号获取或者创建用户信息
	user, err := u.svc.FindOrCreateUser(ctx, req.Phone) // 根据手机号获取用户信息
	if err != nil {
		ctx.JSON(500, gin.H{"error": "获取用户信息失败"})
		return
	}

	// 设置JWT令牌
	err = u.SetJWT(ctx, user.ID)
	if err != nil {
		ctx.JSON(500, gin.H{"error": "设置JWT失败"})
		return
	}
}

// 设置JWT令牌
func (u *UserHandler) SetJWT(ctx *gin.Context, uid int64) error {
	// 生成JWT令牌
	claims := jwt.MapClaims{
		"id":  uid,                                   // 用户ID
		"exp": jwt.TimeFunc().Add(time.Hour * 24).Unix(), // 过期时间为24小时
		"iat": jwt.TimeFunc().Unix(),                     // 签发时间
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("my_secret_key"))
	if err != nil {
		ctx.JSON(500, gin.H{"error": "生成JWT失败"})
		return err
	}
	ctx.Header("jwt", tokenString) // 将JWT令牌添加到响应头中

	ctx.JSON(200, gin.H{"message": "登录成功"}) // 返回登录成功的响应
	return nil
}

