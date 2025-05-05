package cache

import (
	"context"
	"fmt"
	"strconv"

	_ "embed"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/redis/go-redis/v9"
)

//go:embed lua/interaction_incr.lua
var luaIncrCnt string

const (
	fieldViewCount    = "view_count"
	fieldLikeCount    = "like_count"
	fieldCollectCount = "collect_count"
)

type InteractionCacheInterface interface {
	// IncrViewCntIfPresent increments the view count if the interaction exists
	IncrViewCntIfPresent(ctx context.Context, biz string, bizId int64) error
	IncrLikeCntIfPresent(ctx context.Context, biz string, bizId int64) error
	IncrCollectCntIfPresent(ctx context.Context, biz string, bizId int64) error
	DecrLikeCntIfPresent(ctx context.Context, biz string, bizId int64) error
	Get(ctx context.Context, biz string, bizId int64) (domain.Interaction, error)
	Set(ctx context.Context, biz string, bizId int64, interaction domain.Interaction) error
	DecrCollectCntIfPresent(ctx context.Context, biz string, bizId int64) error
	BatchIncrViewCntIfPresent(ctx context.Context, biz []string, bizIds []int64) error
}

type RedisInteractionCache struct {
	client redis.Cmdable
}

func NewRedisInteractionCache(client redis.Cmdable) InteractionCacheInterface {
	return &RedisInteractionCache{
		client: client,
	}
}
func (r *RedisInteractionCache) Key(biz string, bizId int64) string {
	return "interaction:" + biz + ":" + strconv.FormatInt(bizId, 10)
}

// 增加浏览量
func (r *RedisInteractionCache) IncrViewCntIfPresent(ctx context.Context, biz string, bizId int64) error {
	return r.client.Eval(ctx, luaIncrCnt, []string{r.Key(biz, bizId)}, fieldViewCount, 1).Err()
}

// 增加点赞量
func (r *RedisInteractionCache) IncrLikeCntIfPresent(ctx context.Context, biz string, bizId int64) error {
	return r.client.Eval(ctx, luaIncrCnt, []string{r.Key(biz, bizId)}, fieldLikeCount, 1).Err()
}

// 增加收藏量
func (r *RedisInteractionCache) IncrCollectCntIfPresent(ctx context.Context, biz string, bizId int64) error {
	return r.client.Eval(ctx, luaIncrCnt, []string{r.Key(biz, bizId)}, fieldCollectCount, 1).Err()
}

// 减少点赞量
func (r *RedisInteractionCache) DecrLikeCntIfPresent(ctx context.Context, biz string, bizId int64) error {
	return r.client.Eval(ctx, luaIncrCnt, []string{r.Key(biz, bizId)}, fieldLikeCount, -1).Err()
}

// 减少收藏量
func (r *RedisInteractionCache) DecrCollectCntIfPresent(ctx context.Context, biz string, bizId int64) error {
	return r.client.Eval(ctx, luaIncrCnt, []string{r.Key(biz, bizId)}, fieldCollectCount, -1).Err()
}

func (r *RedisInteractionCache) Get(ctx context.Context, biz string, bizId int64) (domain.Interaction, error) {
	// 使用 HGetAll 命令从 Redis 中获取指定键的哈希表中的所有字段和值。
	// r.Key(biz, bizId) 用于生成 Redis 中的键名，标识特定业务和业务 ID 的交互信息。
	data, err := r.client.HGetAll(ctx, r.Key(biz, bizId)).Result()
	if err != nil {
		if err == redis.Nil {
			// 如果 Redis 中没有该键，则返回一个空的 Interaction 对象和 nil 错误
			return domain.Interaction{}, nil
		}
		return domain.Interaction{}, err
	}

	if len(data) == 0 {
		return domain.Interaction{}, fmt.Errorf("interaction not found for biz: %s, bizId: %d in cache", biz, bizId)
	}

	// 解析 Redis 中的字段值
	collectCount, _ := strconv.ParseInt(data[fieldCollectCount], 10, 64)
	likeCount, _ := strconv.ParseInt(data[fieldLikeCount], 10, 64)
	viewCount, _ := strconv.ParseInt(data[fieldViewCount], 10, 64)

	// 将 Redis 中的字段转换为 Interaction 对象
	interaction := domain.Interaction{
		CollectCnt: collectCount,
		LikeCnt:    likeCount,
		ViewCnt:    viewCount,
	}
	return interaction, nil
}

// 设置交互信息
func (r *RedisInteractionCache) Set(ctx context.Context, biz string, bizId int64, interaction domain.Interaction) error {
	// 哈希表的字段包括收藏计数、点赞计数和浏览计数，对应的值从 interaction 结构体中获取。
	err := r.client.HSet(ctx, r.Key(biz, bizId), map[string]any{
		fieldCollectCount: interaction.CollectCnt,
		fieldLikeCount:    interaction.LikeCnt,
		fieldViewCount:    interaction.ViewCnt,
	}).Err()

	if err != nil {

		return err
	}
	return r.client.Expire(ctx, r.Key(biz, bizId), 0).Err()
}

// 批量增加浏览量
func (r *RedisInteractionCache) BatchIncrViewCntIfPresent(ctx context.Context, biz []string, bizIds []int64) error {
	if len(biz) != len(bizIds) {
		return fmt.Errorf("biz and bizIds length mismatch")
	}

	if len(biz) == 0 {
		return nil
	}

	// 使用管道化操作来批量增加浏览量
	pipeline:= r.client.Pipeline()
	for i := 0; i < len(biz); i++ {
		// 使用 Eval 命令执行 Lua 脚本，增加浏览量
		pipeline.Eval(ctx, luaIncrCnt, []string{r.Key(biz[i], bizIds[i])}, fieldViewCount, 1)
	}
	// 执行管道中所有排队的命令，将结果一次性返回
	_, err := pipeline.Exec(ctx)
	return err
}