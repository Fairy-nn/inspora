package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/repository/cache"
	"github.com/Fairy-nn/inspora/internal/repository/dao"
	"github.com/redis/go-redis/v9"
)

// FeedRepository 定义了 Feed 相关的存储操作
type FeedRepository interface {
	// AddToInbox 将一个 Feed 项添加到用户的收件箱
	AddToInbox(ctx context.Context, userID int64, item domain.UserFeedItem) error
	// GetInboxForUser 获取用户的收件箱 Feed (分页)
	GetInboxForUser(ctx context.Context, userID int64, offset, limit int) ([]domain.UserFeedItem, error)
	// TrimInbox 整理用户的收件箱，例如只保留最近的 N 条
	TrimInbox(ctx context.Context, userID int64, maxLength int) error
	// AddFeedEvent 记录一个 Feed 事件
	AddFeedEvent(ctx context.Context, event domain.FeedEvent) (int64, error)
	// AddToOutbox 将用户产生的事件添加到其发件箱 (用于拉模型)
	AddToOutbox(ctx context.Context, userID int64, item domain.UserFeedItem) error
	// GetOutboxForUser 获取用户的发件箱 (分页，用于拉模型)
	GetOutboxForUser(ctx context.Context, userID int64, offset, limit int) ([]domain.UserFeedItem, error)
	// GetFeedEventsSince 获取指定时间之后的所有 Feed 事件 (用于重建 Feed)
	GetFeedEventsSince(ctx context.Context, since time.Time, offset, limit int) ([]domain.FeedEvent, error)
	// GetFeedEventByID 根据 ID 获取 Feed 事件
	GetFeedEventByID(ctx context.Context, id int64) (domain.FeedEvent, error)
}

type CachedFeedRepository struct {
	dao   dao.FeedDAOInterface
	cache cache.FeedCache
}

func NewFeedRepository(dao dao.FeedDAOInterface, cache cache.FeedCache) FeedRepository {
	return &CachedFeedRepository{
		dao:   dao,
		cache: cache,
	}
}
// GetRedisClient 获取 Redis 客户端
func (r *CachedFeedRepository) GetRedisClient() redis.Cmdable {
	return r.cache.GetClient()
}

// AddToInbox 将一个 Feed 项添加到用户的收件箱
func (r *CachedFeedRepository) AddToInbox(ctx context.Context, userID int64, item domain.UserFeedItem) error {
	return r.cache.AddToInbox(ctx, userID, item)
}

// GetInboxForUser 获取用户的收件箱 Feed (分页)
func (r *CachedFeedRepository) GetInboxForUser(ctx context.Context, userID int64, offset, limit int) ([]domain.UserFeedItem, error) {
	return r.cache.GetInboxForUser(ctx, userID, offset, limit)
}

// TrimInbox 整理用户的收件箱，例如只保留最近的 N 条
func (r *CachedFeedRepository) TrimInbox(ctx context.Context, userID int64, maxLength int) error {
	return r.cache.TrimInbox(ctx, userID, maxLength)
}

// AddFeedEvent 记录一个 Feed 事件
func (r *CachedFeedRepository) AddFeedEvent(ctx context.Context, event domain.FeedEvent) (int64, error) {
	// 先持久化到数据库
	id, err := r.dao.SaveFeedEvent(ctx, event)
	if err != nil {
		return 0, fmt.Errorf("保存到数据库失败: %w", err)
	}

	// 更新事件ID
	event.ID = id

	// 再保存到缓存
	_, err = r.cache.AddFeedEvent(ctx, event)
	if err != nil {
		// 这里我们只记录错误但不中断，因为数据已经持久化到数据库中
		fmt.Printf("保存Feed事件到缓存失败: %v\n", err)
	}

	return id, nil
}

// AddToOutbox 将用户产生的事件添加到其发件箱 (用于拉模型)
func (r *CachedFeedRepository) AddToOutbox(ctx context.Context, userID int64, item domain.UserFeedItem) error {
	return r.cache.AddToOutbox(ctx, userID, item)
}

// GetOutboxForUser 获取用户的发件箱 (分页，用于拉模型)
func (r *CachedFeedRepository) GetOutboxForUser(ctx context.Context, userID int64, offset, limit int) ([]domain.UserFeedItem, error) {
	return r.cache.GetOutboxForUser(ctx, userID, offset, limit)
}

// GetFeedEventsSince 获取指定时间之后的所有 Feed 事件 (用于重建 Feed)
func (r *CachedFeedRepository) GetFeedEventsSince(ctx context.Context, since time.Time, offset, limit int) ([]domain.FeedEvent, error) {
	// 直接从数据库获取，因为我们是在重建Feed，不需要从缓存获取
	dbEvents, err := r.dao.GetFeedEventsSince(ctx, since, offset, limit)
	if err != nil {
		return nil, err
	}

	// 转换为领域模型
	result := make([]domain.FeedEvent, 0, len(dbEvents))
	for _, dbEvent := range dbEvents {
		var content map[string]interface{}
		if err := json.Unmarshal([]byte(dbEvent.Content), &content); err != nil {
			continue // 跳过损坏的数据
		}

		result = append(result, domain.FeedEvent{
			ID:        dbEvent.ID,
			UserID:    dbEvent.UserID,
			EventType: dbEvent.EventType,
			Content:   content,
			Ctime:     dbEvent.Ctime,
		})
	}

	return result, nil
}

// GetFeedEventByID 根据 ID 获取 Feed 事件
func (r *CachedFeedRepository) GetFeedEventByID(ctx context.Context, id int64) (domain.FeedEvent, error) {
	// 直接从数据库获取，因为我们是在重建Feed，不需要从缓存获取
	dbEvent, err := r.dao.GetFeedEventByID(ctx, id)
	if err != nil {
		return domain.FeedEvent{}, err
	}

	// 转换为领域模型
	var content map[string]interface{}
	if err := json.Unmarshal([]byte(dbEvent.Content), &content); err != nil {
		return domain.FeedEvent{}, fmt.Errorf("解析事件内容失败: %w", err)
	}

	return domain.FeedEvent{
		ID:        dbEvent.ID,
		UserID:    dbEvent.UserID,
		EventType: dbEvent.EventType,
		Content:   content,
		Ctime:     dbEvent.Ctime,
	}, nil
}
