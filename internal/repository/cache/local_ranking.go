package cache

import (
	"context"
	"errors"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/ecodeclub/ekit/syncx/atomicx"
)

type LocalRankingCache struct {
	topN       *atomicx.Value[[]domain.Article]
	ddl        *atomicx.Value[time.Time]
	expiration time.Duration
}

func NewLocalRankingCache() *LocalRankingCache {
	return &LocalRankingCache{
		topN:       atomicx.NewValue[[]domain.Article](),
		ddl:        atomicx.NewValueOf(time.Now()),
		expiration: time.Minute * 10,
	}
}

// Set 存入缓存
func (l *LocalRankingCache) Set(ctx context.Context, articles []domain.Article) error {
	l.topN.Store(articles)
	l.ddl.Store(time.Now().Add(l.expiration))
	return nil
}

// Get 获取缓存
func (l *LocalRankingCache) Get(ctx context.Context) ([]domain.Article, error) {
	ddl := l.ddl.Load()
	articles := l.topN.Load()

	if len(articles) == 0 || ddl.Before(time.Now()) {
		return nil, errors.New("cache expired")
	}
	return articles, nil
}

// ForceGet 强制获取缓存,不检查过期时间
func (l *LocalRankingCache) ForceGet(ctx context.Context) ([]domain.Article, error) {
	articles := l.topN.Load()
	if len(articles) == 0 {
		return nil, errors.New("cache expired")
	}
	return articles, nil
}
