package repository

import (
	"context"
	"fmt"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/repository/cache"
)

type RankingRepositoryInterface interface {
	ReplaceTopN(ctx context.Context, articles []domain.Article) error
	GetTopN(ctx context.Context) ([]domain.Article, error)
}

type CachedRankingRepository struct {
	// 这里的缓存可以是redis, memcached, local
	redis cache.RankingCacheInterface
	local cache.LocalRankingCache
}

func NewCachedRankingRepository(redis cache.RankingCacheInterface, local cache.LocalRankingCache) RankingRepositoryInterface {
	return &CachedRankingRepository{
		redis: redis,
		local: local,
	}
}

// ReplaceTopN 替换前N名的文章
func (r *CachedRankingRepository) ReplaceTopN(ctx context.Context, articles []domain.Article) error {
	_ = r.local.Set(ctx, articles)
	return r.redis.Set(ctx, articles)
}

// GetTopN 获取前N名的文章
// 先从本地缓存中获取,如果没有,就从redis中获取,如果redis也没有,就强制获取
func (r *CachedRankingRepository) GetTopN(ctx context.Context) ([]domain.Article, error) {
	fmt.Printf("GetTopN: %s\n", "get top n")
	// 先从本地缓存中获取
	data, err := r.local.Get(ctx)
	fmt.Printf("local_err: %v\n", err)
	fmt.Printf("local_data: %v\n", data)
	if err == nil {
		return data, nil
	}

	// 如果本地缓存没有,就从redis中获取
	data, err = r.redis.Get(ctx)
	fmt.Printf("redis_err: %v\n", err)
	fmt.Printf("redis_data: %v\n", data)
	if err == nil {
		r.local.Set(ctx, data)
	} else {
		fmt.Printf("force_err: %v\n", err)
		fmt.Printf("force_data: %v\n", data)
		return r.local.ForceGet(ctx)
	}
	return data, err
}
