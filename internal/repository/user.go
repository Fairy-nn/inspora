package repository

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"sync"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/repository/cache"
	"github.com/Fairy-nn/inspora/internal/repository/dao"
)

type UserRepositoryInterface interface {
	Create(ctx context.Context, u domain.User) error
	GetByPhone(ctx context.Context, phone string) (domain.User, error)
	GetByID(ctx context.Context, id int64) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
}

type UserRepository struct {
	dao   dao.UserDaoInterface
	cache cache.UserCacheInterface
}

func NewUserRepository(dao dao.UserDaoInterface, cache cache.UserCacheInterface) UserRepositoryInterface {
	return &UserRepository{
		dao:   dao,
		cache: cache,
	}
}

var (
	errUserNotFound       = errors.New("用户不存在")
	ErrUserNotFound       = dao.ErrUserNotFound
	ErrUserDuplicateEmail = dao.ErrUserDuplicateEmail
)

// Create 创建用户
func (repo *UserRepository) Create(ctx context.Context, u domain.User) error {
	userEntity := repo.domainToEntity(u)
	return repo.dao.Insert(ctx, &userEntity) // 插入用户数据到数据库
	// TODO:缓存
}

// GetByEmail 根据邮箱查找用户
func (repo *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	user, err := repo.dao.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, dao.ErrUserNotFound) {
			return domain.User{}, errUserNotFound // 用户不存在
		}
		return domain.User{}, err // 其他错误
	}
	// 返回用户信息
	return repo.enityToDomain(*user), nil
}

var mutex sync.Mutex

// GetByID 根据用户ID获取用户信息
func (r *UserRepository) GetByID(ctx context.Context, id int64) (domain.User, error) {
	user, err := r.cache.Get(ctx, id)
	if err == nil { // 如果缓存中存在用户信息，直接返回
		return user, nil
	}
	// 在这里加锁，确保只有一个协程在执行数据库查询，避免缓存击穿
	mutex.Lock()
	defer mutex.Unlock()

	// 如果缓存中不存在用户信息，从数据库中获取
	daoUser, err := r.dao.GetByID(ctx, id)
	if err != nil {
		return domain.User{}, err
	}

	// 将数据库中的用户信息转换为domain.User类型
	user = r.enityToDomain(daoUser)

	// 将用户信息存入缓存
	// 这里避免缓存失败，使用重试机制
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err = r.cache.Set(ctx, user)
		if err == nil {
			break
		}
		log.Printf("Failed to set user in cache (attempt %d): %v", i+1, err)
	}
	if err != nil {
		log.Printf("Failed to set user in cache after %d attempts: %v", maxRetries, err)
	}
	return user, nil
}

// 根据手机号获取用户信息
func (r *UserRepository) GetByPhone(ctx context.Context, phone string) (domain.User, error) {
	user, err := r.dao.GetByPhone(ctx, phone)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			//fmt.Println("用户不存在%+v", err) // DEBUG: 打印用户不存在的提示
			return domain.User{}, ErrUserNotFound // 用户不存在
		}
		//fmt.Println("获取用户信息失败%+v", err) // DEBUG: 打印获取用户信息失败的提示
		return domain.User{}, err // 其他错误
	}
	// 返回用户信息
	return r.enityToDomain(user), nil
}

// 将dao.User转换为domain.User
func (r *UserRepository) enityToDomain(u dao.User) domain.User {
	return domain.User{
		ID:       u.ID,
		Email:    u.Email.String,
		Phone:    u.Phone.String,
		Username: u.Username,
		Password: u.Password,
	}
}

// 将domain.User转换为dao.User
func (r *UserRepository) domainToEntity(u domain.User) dao.User {
	return dao.User{
		ID:       u.ID,
		Email:    sql.NullString{String: u.Email, Valid: u.Email != ""},
		Username: u.Username,
		Password: u.Password,
		Phone:    sql.NullString{String: u.Phone, Valid: u.Phone != ""},
	}
}
