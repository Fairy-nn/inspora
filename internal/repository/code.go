package repository

import (
	"context"
	"fmt"

	"github.com/Fairy-nn/inspora/internal/repository/cache"
)

var (
	ErrCodeSentTooManyTimes  = cache.ErrCodeSentTomanyTimes
	ErrCodeTriedTooManyTimes = cache.ErrCodeTriesTooMany
)

type CodeRepository struct {
	cache *cache.CodeCache
}

func NewCodeRepository(cache *cache.CodeCache) *CodeRepository {
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
		return fmt.Errorf("验证码存储失败le: %w", err)
	}
	return nil
}

func (r *CodeRepository) Verify(ctx context.Context, biz, phone, code string) (bool, error) {
	result, err := r.cache.Verify(ctx, biz, phone, code)
	if err != nil {
		return false, fmt.Errorf("验证码验证失败: %w", err)
	}
	return result, nil
}
