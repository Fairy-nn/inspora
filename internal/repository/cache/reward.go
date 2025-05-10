package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/redis/go-redis/v9"
)

type RewardCacheInterface interface {
	GetCachedCodeURL(ctx context.Context, r domain.Reward) (domain.CodeURL, error) // 获取缓存的二维码URL
	CachedCodeURL(ctx context.Context, cu domain.CodeURL, r domain.Reward) error   // 缓存二维码URL
}

// 基于Redis的打赏二维码缓存实现
type RewardRedisCache struct {
	client redis.Cmdable // Redis客户端接口，支持多种Redis实现
}

func NewRewardRedisCache(client redis.Cmdable) RewardCacheInterface {
	return &RewardRedisCache{client: client}
}

// 生成二维码URL的Redis键
// 格式: reward:code_url:{业务类型}:{业务ID}:{用户ID}
func (c *RewardRedisCache) codeURLKey(r domain.Reward) string {
	return fmt.Sprintf("reward:code_url:%s:%d:%d", r.Target.Biz, r.Target.BizId, r.UserID)
}

// 从Redis中获取缓存的二维码URL
func (c *RewardRedisCache) GetCachedCodeURL(ctx context.Context, r domain.Reward) (domain.CodeURL, error) {
	key := c.codeURLKey(r)
	val, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return domain.CodeURL{}, err // 可能是redis.Nil(缓存未命中)或其他错误
	}

	var res domain.CodeURL
	err = json.Unmarshal(val, &res) // 反序列化JSON数据
	return res, err
}

// 将二维码URL缓存到Redis
func (c *RewardRedisCache) CachedCodeURL(ctx context.Context, cu domain.CodeURL, r domain.Reward) error {
	key := c.codeURLKey(r)
	val, err := json.Marshal(cu) // 序列化二维码URL信息为JSON
	if err != nil {
		return err
	}

	// 存入Redis，设置29分钟过期时间
	// 略小于30分钟，错开与订单过期检查的周期，避免缓存雪崩
	return c.client.Set(ctx, key, val, time.Minute*29).Err()
}
