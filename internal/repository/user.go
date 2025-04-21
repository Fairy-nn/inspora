package repository

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/repository/cache"
	"github.com/Fairy-nn/inspora/internal/repository/dao"
)

type UserRepository struct {
	dao   *dao.UserDAO
	cache *cache.UserCache
}

func NewUserRepository(dao *dao.UserDAO, cache *cache.UserCache) *UserRepository {
	return &UserRepository{
		dao:   dao,
		cache: cache,
	}
}

var (
	errUserNotFound = errors.New("用户不存在")
)

// Create 创建用户
func (repo *UserRepository) Create(ctx context.Context, u domain.User) error {
	return repo.dao.Insert(ctx, &dao.User{
		Email:    u.Email,
		Password: u.Password,
		Username: u.Username,
	})
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
	return domain.User{
		Email:    user.Email,
		Password: user.Password,
		ID:       user.ID,
	}, nil
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
	user = domain.User{
		ID:       daoUser.ID,
		Email:    daoUser.Email,
		Username: daoUser.Username,
		Password: daoUser.Password,
	}
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
