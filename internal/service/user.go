package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/repository"
	"github.com/Fairy-nn/inspora/internal/repository/cache"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UserServiceInterface interface {
	SignUp(ctx *gin.Context, u domain.User) error
	Login(ctx *gin.Context, u domain.User) (domain.User, error)
	Profile(ctx context.Context, userID int64) (domain.User, error)
	FindOrCreateUser(ctx *gin.Context, phone string) (domain.User, error)
}

// UserService 用户服务结构体
type UserService struct {
	repo repository.UserRepositoryInterface // 用户存储库接口

}

func NewUserService(repo repository.UserRepositoryInterface) UserServiceInterface {
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
	fmt.Printf("user:%+v", user) // DEBUG: 打印用户信息
	if err != nil {              // 如果没有找到用户，返回错误
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

// Profile 获取用户信息
func (svc *UserService) Profile(ctx context.Context, userID int64) (domain.User, error) {
	user, err := svc.repo.GetByID(ctx, userID)
	if err != nil {
		if err == cache.ErrKeyNotExist {
			return domain.User{}, err
		}
		return domain.User{}, err
	}
	return user, nil
}

// 获取或者创建用户
// 通过手机号获取用户信息
func (u *UserService) FindOrCreateUser(ctx *gin.Context, phone string) (domain.User, error) {
	// 先尝试获取用户信息

	user, err := u.repo.GetByPhone(ctx, phone)
	if err == nil {
		//fmt.Printf("用户信息已存在 %+v\n", user) // DEBUG: 打印用户信息
		return user, nil // 如果存在用户信息，直接返回
	}
	// 如果用户存在但不是预期的错误，返回错误
	if err != repository.ErrUserNotFound {
		//fmt.Printf("获取用户信息失败sssss: %v\n", err) // DEBUG: 打印错误信息
		return domain.User{}, err
	}

	// 创建新用户
	err = u.repo.Create(ctx, domain.User{
		Phone: phone,
	})
	// 如果创建用户时发生错误，检查是否是重复条目错误
	if err != nil && err != repository.ErrUserDuplicateEmail {

		return domain.User{}, err
	}
	return u.repo.GetByPhone(ctx, phone)
}
