package repository

import (
	"context"

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
	return r.cache.Set(ctx, biz, phone, code)
}
