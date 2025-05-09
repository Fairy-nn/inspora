package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/redis/go-redis/v9"
)

type RankingCacheInterface interface {
	Set(ctx context.Context, articles []domain.Article) error
	Get(ctx context.Context) ([]domain.Article, error)
}

type RedisRankingCache struct {
	client redis.Cmdable
	key    string
}

func NewRedisRankingCache(client redis.Cmdable) *RedisRankingCache {
	return &RedisRankingCache{
		client: client,
		key:    "ranking",
	}
}

// SET存入缓存
func (r *RedisRankingCache) Set(ctx context.Context, articles []domain.Article) error {
	for i := 0; i < len(articles); i++ {
		articles[i].Content = ""
	}
	val, err := json.Marshal(articles)
	if err != nil {

		return err
	}
	return r.client.Set(ctx, r.key, val, time.Minute*10).Err()
}

func (r *RedisRankingCache) Get(ctx context.Context) ([]domain.Article, error) {
	data, err := r.client.Get(ctx, r.key).Bytes()
	if err != nil {
		return nil, err
	}

	var res []domain.Article
	json.Unmarshal(data, &res)
	return res, err
}