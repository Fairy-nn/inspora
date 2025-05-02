package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/redis/go-redis/v9"
)

type ArticleCache interface {
	// GetArticleList 从缓存中获取文章列表
	GetFirstPage(ctx context.Context, uid int64) ([]domain.Article, error)
	// GetArticleList 缓存第一页文章到redis中
	SetFirstPage(ctx context.Context, uid int64, articles []domain.Article) error
	// Set 将单篇文章缓存到redis中
	Set(ctx context.Context, article domain.Article, uid int64) error
	// DelFirstPage 删除缓存
	DelFirstPage(ctx context.Context, uid int64) error
	// SetPub 设置发布文章的缓存
	SetPub(ctx context.Context, article domain.Article) error
}

type RedisArticleCache struct {
	client redis.Cmdable
}

func NewRedisArticleCache(client redis.Cmdable) ArticleCache {
	return &RedisArticleCache{client: client}
}

// GetFirstPage 从缓存中获取文章列表
func (a *RedisArticleCache) GetFirstPage(ctx context.Context, uid int64) ([]domain.Article, error) {
	// 根据用户 ID 生成的key从 Redis 中获取缓存数据
	data, err := a.client.Get(ctx, a.KeyList(uid)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	// 存储反序列化数据
	var articles []domain.Article
	if err := json.Unmarshal([]byte(data), &articles); err != nil {
		return nil, err
	}

	return articles, nil
}

// SetFirstPage 缓存第一页文章到redis中
func (a *RedisArticleCache) SetFirstPage(ctx context.Context, uid int64, articles []domain.Article) error {
	// 将文章内容替换成摘要
	for i := 0; i < len(articles); i++ {
		articles[i].Content = articles[i].GenerateAbstract()
	}

	// Redis 存储的数据通常是字节切片，Redis 存储的数据通常是字节切片
	data, err := json.Marshal(articles)
	if err != nil {
		return err
	}
	// 设置缓存，key为文章ID，值为序列化后的文章对象，过期时间为10分钟
	return a.client.Set(ctx, a.KeyList(uid), data, time.Minute*10).Err()
}
// KeyArticle 生成文章缓存的key
func (r *RedisArticleCache) KeyArticle(uid int64) string {
	return fmt.Sprintf("article:first_article:%d", uid)
}
// KeyList 生成文章列表缓存的key
func (r *RedisArticleCache) KeyList(uid int64) string {
	return fmt.Sprintf("article:first_page:%d", uid)
}
// keyArticlePub 生成发布文章缓存的key
func (r *RedisArticleCache) KeyArticlePub(id int64) string {
	return fmt.Sprintf("article:pub:%d", id)
}

// Set 将单篇文章缓存到redis中
func (a *RedisArticleCache) Set(ctx context.Context, article domain.Article, uid int64) error {
	// 序列化文章
	data, err := json.Marshal(article)
	if err != nil {
		return err
	}

	// 设置缓存，key为文章ID，值为序列化后的文章对象，过期时间为15秒
	return a.client.Set(ctx, a.KeyArticle(uid), data, time.Second*15).Err()
}

// DelFirstPage 删除缓存
func (a *RedisArticleCache) DelFirstPage(ctx context.Context, uid int64) error {
	return a.client.Del(ctx, a.KeyList(uid)).Err()
}

// SetPub 设置发布文章的缓存
func (a *RedisArticleCache) SetPub(ctx context.Context, article domain.Article) error {
	// 序列化文章
	data, err := json.Marshal(article)
	if err != nil {
		return err
	}

	// 设置缓存，key为文章ID，值为序列化后的文章对象，过期时间为15秒
	return a.client.Set(ctx, a.KeyArticlePub(article.ID), data, time.Second*15).Err()
}
