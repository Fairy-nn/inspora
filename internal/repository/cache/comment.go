package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Fairy-nn/inspora/internal/repository/dao"
	"github.com/redis/go-redis/v9"
)

// 缓存三种类型的评论：
//	文章的前三条评论
//	文章的热门评论
// 热门评论的子评论

const (
	// 评论缓存前缀
	commentCacheKeyPrefix = "comment:"
	// 评论列表缓存前缀
	commentListCacheKeyPrefix = "comment:list:"
	// 热门评论缓存前缀
	hotCommentCacheKeyPrefix = "comment:hot:"
)

type CommentCache interface {
	// GetComment 获取评论缓存
	GetComment(ctx context.Context, id int64) (dao.Comment, error)
	// SetComment 设置评论缓存
	SetComment(ctx context.Context, comment dao.Comment, expiration time.Duration) error
	// DelComment 删除评论缓存
	DelComment(ctx context.Context, id int64) error

	// GetHotComments 获取热门评论缓存
	GetHotComments(ctx context.Context, biz string, bizID int64) ([]dao.Comment, error)
	// SetHotComments 设置热门评论缓存
	SetHotComments(ctx context.Context, biz string, bizID int64, comments []dao.Comment, expiration time.Duration) error
	// DelHotComments 删除热门评论缓存
	DelHotComments(ctx context.Context, biz string, bizID int64) error

	// PreloadComments 预加载文章前三条评论及其子评论
	PreloadComments(ctx context.Context, biz string, bizID int64, comments []dao.Comment) error
}

type RedisCommentCache struct {
	client redis.Cmdable
}

func NewRedisCommentCache(client redis.Cmdable) CommentCache {
	return &RedisCommentCache{
		client: client,
	}
}

// getCommentKey 生成评论缓存的键
func getCommentKey(id int64) string {
	return fmt.Sprintf("%s%d", commentCacheKeyPrefix, id)
}

// getHotCommentKey 生成热门评论缓存键
func getHotCommentKey(biz string, bizID int64) string {
	return fmt.Sprintf("%s%s:%d", hotCommentCacheKeyPrefix, biz, bizID)
}

// GetComment 获取评论缓存
func (r *RedisCommentCache) GetComment(ctx context.Context, id int64) (dao.Comment, error) {
	// 生成评论缓存的键
	key := getCommentKey(id)
	// 从 Redis 中获取评论缓存
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return dao.Comment{}, err
	}
	// 反序列化评论对象
	var comment dao.Comment
	err = json.Unmarshal([]byte(val), &comment)
	return comment, err
}

// SetComment 设置评论缓存
func (r *RedisCommentCache) SetComment(ctx context.Context, comment dao.Comment, expiration time.Duration) error {
	// 生成评论缓存的键
	key := getCommentKey(comment.ID)
	val, err := json.Marshal(comment)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, val, expiration).Err()
}

// DelComment 删除评论缓存
func (r *RedisCommentCache) DelComment(ctx context.Context, id int64) error {
	// 生成评论缓存的键
	key := getCommentKey(id)
	return r.client.Del(ctx, key).Err()
}

// GetHotComments 获取热门评论缓存
func (r *RedisCommentCache) GetHotComments(ctx context.Context, biz string, bizID int64) ([]dao.Comment, error) {
	// 生成热门评论缓存的键
	key := getHotCommentKey(biz, bizID)
	// 从 Redis 中获取热门评论缓存
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	// 反序列化热门评论对象
	var comments []dao.Comment
	err = json.Unmarshal([]byte(val), &comments)
	return comments, err
}

// SetHotComments 设置热门评论缓存
func (r *RedisCommentCache) SetHotComments(ctx context.Context, biz string, bizID int64, comments []dao.Comment, expiration time.Duration) error {
	// 生成热门评论缓存的键
	key := getHotCommentKey(biz, bizID)
	val, err := json.Marshal(comments)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, val, expiration).Err()
}

// DelHotComments 删除热门评论缓存
func (r *RedisCommentCache) DelHotComments(ctx context.Context, biz string, bizID int64) error {
	// 生成热门评论缓存的键
	key := getHotCommentKey(biz, bizID)
	return r.client.Del(ctx, key).Err()
}

// PreloadComments 预加载文章前三条评论及其子评论
func (r *RedisCommentCache) PreloadComments(ctx context.Context, biz string, bizID int64, comments []dao.Comment) error {
	for _, comment := range comments {
		// 单独缓存每条评论，用户可能只查看某条评论的详情（如点击 "查看完整评论"）
		if err := r.SetComment(ctx, comment, time.Hour*24); err != nil {
			return err
		}
	}
	// 将热门评论列表缓存到 Redis
	return r.SetHotComments(ctx, biz, bizID, comments, time.Hour*24);
}
