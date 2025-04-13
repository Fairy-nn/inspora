package service

import (
	"errors"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/repository"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// UserService 用户服务结构体
type UserService struct {
	repo *repository.UserRepository // 用户存储库接口

}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{
		repo: repo,
	}
}

// SignUp 注册用户
func (svc *UserService) SignUp(ctx *gin.Context, u domain.User) error {
	// 使用 bcrypt 对密码进行哈希处理
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	// 调用存储库的 Create 方法创建用户
	return svc.repo.Create(ctx, u)
}

var (
	errInvalidCredentials = errors.New("密码或邮箱不正确")
	errUserNotFound       = errors.New("用户不存在")
)

// Login 用户登录
func (svc *UserService) Login(ctx *gin.Context, u domain.User) (domain.User, error) {
	// 根据邮箱查找用户
	user, err := svc.repo.GetByEmail(ctx, u.Email)
	if err != nil { // 如果没有找到用户，返回错误
		if errors.Is(err, errUserNotFound) {
			return domain.User{}, errUserNotFound
		}
		return domain.User{}, err // 其他错误
	}

	// 使用 bcrypt 验证密码
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(u.Password)) // 验证密码
	if err != nil {
		// DEBUG: 这里可以打印错误信息
		return domain.User{}, errInvalidCredentials
	}

	// 登录逻辑
	return user, nil
}
