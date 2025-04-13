package repository

import (
	"context"
	"errors"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/repository/dao"
)

type UserRepository struct {
	dao *dao.UserDAO
}

func NewUserRepository(dao *dao.UserDAO) *UserRepository {
	return &UserRepository{
		dao: dao,
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
	}, nil
}
