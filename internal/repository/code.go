package repository

import (
	"context"
	"fmt"

	"github.com/Fairy-nn/inspora/internal/repository/cache"
)

var (
	ErrCodeSentTooManyTimes  = cache.ErrCodeSentTomanyTimes
	ErrCodeTriedTooManyTimes = cache.ErrCodeTriesTooMany
	ErrCodeNotExpired        = cache.ErrCodeNotExpired
)

type CodeRepositoryInterface interface {
	Store(ctx context.Context, biz, phone, code string) error
	Verify(ctx context.Context, biz, phone, code string) (bool, error)
}

type CodeRepository struct {
	cache cache.CodeCacheInterface
}

func NewCodeRepository(cache cache.CodeCacheInterface) CodeRepositoryInterface {
	return &CodeRepository{
		cache: cache,
	}
}

func (r *CodeRepository) Store(ctx context.Context, biz, phone, code string) error {
	// 生成验证码并存储到缓存中
	// 打印phone和code
	fmt.Printf("phone: %s, code: %s\n", phone, code) // DEBUG: 打印验证码信息
	err := r.cache.Set(ctx, biz, phone, code)
	if err != nil {
		if err == cache.ErrCodeNotExpired {
			return err
		}else if err == cache.ErrCodeSentTomanyTimes {
			return err
		}
		return fmt.Errorf("验证码存储失败: %w", err) // 返回错误	
	}
	return nil
}

func (r *CodeRepository) Verify(ctx context.Context, biz, phone, code string) (bool, error) {
	result, err := r.cache.Verify(ctx, biz, phone, code)
	if err != nil {
		return false, err // 返回错误
	}
	return result, nil
}
