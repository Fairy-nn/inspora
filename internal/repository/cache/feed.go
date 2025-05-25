package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/redis/go-redis/v9"
)

const (
	// Redis keys
	userInboxKeyPrefix  = "feed:inbox:"  // 用户收件箱前缀
	userOutboxKeyPrefix = "feed:outbox:" // 用户发件箱前缀
	feedEventsKey       = "feed:events"  // Feed 事件时间线
	feedEventPrefix     = "feed:event:"  // Feed 事件详情前缀
)

type FeedCache interface {
	AddToInbox(ctx context.Context, userID int64, item domain.UserFeedItem) error
	GetInboxForUser(ctx context.Context, userID int64, offset, limit int) ([]domain.UserFeedItem, error)
	TrimInbox(ctx context.Context, userID int64, maxLength int) error
	AddFeedEvent(ctx context.Context, event domain.FeedEvent) (int64, error)
	AddToOutbox(ctx context.Context, userID int64, item domain.UserFeedItem) error
	GetOutboxForUser(ctx context.Context, userID int64, offset, limit int) ([]domain.UserFeedItem, error)
	GetFeedEventsSince(ctx context.Context, since time.Time, offset, limit int) ([]domain.FeedEvent, error)
	GetClient() redis.Cmdable
}

// RedisFeedCache 基于 Redis 实现的 FeedRepository
type RedisFeedCache struct {
	client redis.Cmdable
}

// NewRedisFeedCache 创建一个新的 RedisFeedCache
func NewRedisFeedCache(client redis.Cmdable) FeedCache {
	return &RedisFeedCache{
		client: client,
	}
}

// AddToInbox 将一个 Feed 项添加到用户的收件箱
func (r *RedisFeedCache) AddToInbox(ctx context.Context, userID int64, item domain.UserFeedItem) error {
	key := fmt.Sprintf("%s%d", userInboxKeyPrefix, userID)
	value, err := json.Marshal(item)
	if err != nil {
		return err
	}

	// 使用 ZADD 将 Feed 项添加到用户的收件箱，以时间戳为分数进行排序
	score := float64(item.Timestamp.UnixNano())
	return r.client.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: value,
	}).Err()
}

// GetInboxForUser 获取用户的收件箱 Feed (分页)
func (r *RedisFeedCache) GetInboxForUser(ctx context.Context, userID int64, offset, limit int) ([]domain.UserFeedItem, error) {
	key := fmt.Sprintf("%s%d", userInboxKeyPrefix, userID)

	// 使用 ZREVRANGE 获取按时间戳降序排列的 Feed 项
	result, err := r.client.ZRevRange(ctx, key, int64(offset), int64(offset+limit-1)).Result()
	if err != nil {
		return nil, err
	}

	items := make([]domain.UserFeedItem, 0, len(result))
	for _, value := range result {
		var item domain.UserFeedItem
		if err := json.Unmarshal([]byte(value), &item); err != nil {
			continue // 跳过损坏的数据
		}
		items = append(items, item)
	}

	return items, nil
}

// TrimInbox 整理用户的收件箱，例如只保留最近的 N 条
func (r *RedisFeedCache) TrimInbox(ctx context.Context, userID int64, maxLength int) error {
	key := fmt.Sprintf("%s%d", userInboxKeyPrefix, userID)

	// 获取收件箱大小
	size, err := r.client.ZCard(ctx, key).Result()
	if err != nil {
		return err
	}

	// 如果超出最大长度，删除多余的条目（保留最新的）
	if size > int64(maxLength) {
		return r.client.ZRemRangeByRank(ctx, key, 0, size-int64(maxLength)-1).Err()
	}

	return nil
}

// AddFeedEvent 记录一个 Feed 事件
func (r *RedisFeedCache) AddFeedEvent(ctx context.Context, event domain.FeedEvent) (int64, error) {
	// 首先获取一个新的事件 ID（使用自增）
	eventID, err := r.client.Incr(ctx, "feed:event:id").Result()
	if err != nil {
		return 0, err
	}

	// 设置事件 ID
	event.ID = eventID

	// 序列化事件
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return 0, err
	}

	// 保存事件详情
	eventKey := fmt.Sprintf("%s%d", feedEventPrefix, eventID)
	if err := r.client.Set(ctx, eventKey, eventJSON, 0).Err(); err != nil {
		return 0, err
	}

	// 添加到事件时间线
	score := float64(event.Ctime.UnixNano())
	if err := r.client.ZAdd(ctx, feedEventsKey, redis.Z{
		Score:  score,
		Member: eventID,
	}).Err(); err != nil {
		return 0, err
	}

	return eventID, nil
}

// AddToOutbox 将用户产生的事件添加到其发件箱 (用于拉模型)
func (r *RedisFeedCache) AddToOutbox(ctx context.Context, userID int64, item domain.UserFeedItem) error {
	key := fmt.Sprintf("%s%d", userOutboxKeyPrefix, userID)
	value, err := json.Marshal(item)
	if err != nil {
		return err
	}

	// 使用 ZADD 将 Feed 项添加到用户的发件箱，以时间戳为分数进行排序
	score := float64(item.Timestamp.UnixNano())
	return r.client.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: value,
	}).Err()
}

// GetOutboxForUser 获取用户的发件箱 (分页，用于拉模型)
func (r *RedisFeedCache) GetOutboxForUser(ctx context.Context, userID int64, offset, limit int) ([]domain.UserFeedItem, error) {
	key := fmt.Sprintf("%s%d", userOutboxKeyPrefix, userID)

	// 使用 ZREVRANGE 获取按时间戳降序排列的 Feed 项
	result, err := r.client.ZRevRange(ctx, key, int64(offset), int64(offset+limit-1)).Result()
	if err != nil {
		return nil, err
	}

	items := make([]domain.UserFeedItem, 0, len(result))
	for _, value := range result {
		var item domain.UserFeedItem
		if err := json.Unmarshal([]byte(value), &item); err != nil {
			continue // 跳过损坏的数据
		}
		items = append(items, item)
	}

	return items, nil
}

// GetFeedEventsSince 获取指定时间之后的所有 Feed 事件 (用于重建 Feed)
func (r *RedisFeedCache) GetFeedEventsSince(ctx context.Context, since time.Time, offset, limit int) ([]domain.FeedEvent, error) {
	// 计算时间戳作为分数
	score := float64(since.UnixNano())

	// 使用 ZRANGEBYSCORE 获取指定分数范围的事件 ID
	eventIDs, err := r.client.ZRangeByScore(ctx, feedEventsKey, &redis.ZRangeBy{
		Min:    strconv.FormatFloat(score, 'f', 0, 64),
		Max:    "+inf",
		Offset: int64(offset),
		Count:  int64(limit),
	}).Result()
	if err != nil {
		return nil, err
	}

	events := make([]domain.FeedEvent, 0, len(eventIDs))
	for _, idStr := range eventIDs {
		// 获取事件详情
		id, _ := strconv.ParseInt(idStr, 10, 64)
		eventKey := fmt.Sprintf("%s%d", feedEventPrefix, id)
		eventJSON, err := r.client.Get(ctx, eventKey).Result()
		if err != nil {
			if err == redis.Nil {
				continue // 跳过不存在的事件
			}
			return nil, err
		}

		var event domain.FeedEvent
		if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
			continue // 跳过损坏的数据
		}

		events = append(events, event)
	}

	return events, nil
}

// GetClient 返回 Redis 客户端，供 repository 层使用
func (r *RedisFeedCache) GetClient() redis.Cmdable {
	return r.client
}
