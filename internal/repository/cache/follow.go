package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/redis/go-redis/v9"
)

const (
	// 所有用户的统计信息
	followStatisticsKey = "follow:statistics"
	// 关注关系，每个用户关注的人
	followRelationKeyPrefix = "follow:relation:"
)

type FollowCache interface {
	// GetFolloweeList 获取某人的关注列表
	GetStatistics(ctx context.Context, uid int64) (domain.FollowStatistics, error)
	// SetStatistics 设置某个用户的关注统计信息
	SetStatistics(ctx context.Context, uid int64, statistics domain.FollowStatistics) error
	// AddFollow 添加关注关系
	AddFollow(ctx context.Context, follower, followee int64) error
	// RemoveFollow 移除关注关系
	RemoveFollow(ctx context.Context, follower, followee int64) error
	// IsFollowing 判断某人是否关注了另一个人
	// 这里的follower是关注者，followee是被关注者
	IsFollowing(ctx context.Context, follower, followee int64) (bool, error)
}

type RedisFollowCache struct {
	client redis.Cmdable
}

func NewRedisFollowCache(client redis.Cmdable) FollowCache {
	return &RedisFollowCache{
		client: client,
	}
}

// GetStatistics 获取某个用户的关注统计信息
func (r *RedisFollowCache) GetStatistics(ctx context.Context, uid int64) (domain.FollowStatistics, error) {
	uidStr := strconv.FormatInt(uid, 10) // 将uid转换为字符串

	// 从Redis中获取统计数据
	val, err := r.client.HGet(ctx, followStatisticsKey, uidStr).Result()
	if err != nil {
		return domain.FollowStatistics{}, err
	}

	// 反序列化为结构体
	var res domain.FollowStatistics
	err = json.Unmarshal([]byte(val), &res)
	return res, err
}

// SetStatistics 设置某个用户的关注统计信息
func (r *RedisFollowCache) SetStatistics(ctx context.Context, uid int64, statistics domain.FollowStatistics) error {
	uidStr := strconv.FormatInt(uid, 10) // 将uid转换为字符串

	// 序列化为JSON字符串
	data, err := json.Marshal(statistics)
	if err != nil {
		return err
	}

	// 将数据存储到Redis中
	return r.client.HSet(ctx, followStatisticsKey, uidStr, data).Err()
}

// AddFollow 添加关注关系
func (r *RedisFollowCache) AddFollow(ctx context.Context, follower, followee int64) error {
	// 使用事务确保原子性
	pipe := r.client.TxPipeline()
	followerStr := strconv.FormatInt(follower, 10)
	followeeStr := strconv.FormatInt(followee, 10)
	relationKey := fmt.Sprintf("%s%d", followRelationKeyPrefix, follower)

	// 设置关注关系
	pipe.HSet(ctx, relationKey, followeeStr, 1)
	// 尝试更新关注者的统计数据
	followerStatsCmd := pipe.HGet(ctx, followStatisticsKey, followerStr)
	// 尝试更新被关注者的统计数据
	followeeStatsCmd := pipe.HGet(ctx, followStatisticsKey, followeeStr)

	// 执行事务
	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}

	// 处理关注者统计数据
	followerStatsVal, err := followerStatsCmd.Result()
	if err == nil {
		var stats domain.FollowStatistics
		if err = json.Unmarshal([]byte(followerStatsVal), &stats); err == nil {
			stats.Followees++
			if newVal, err := json.Marshal(stats); err == nil {
				r.client.HSet(ctx, followStatisticsKey, followerStr, newVal)
			}
		}
	}

	// 处理被关注者统计数据
	followeeStatsVal, err := followeeStatsCmd.Result()
	if err == nil {
		var stats domain.FollowStatistics
		if err = json.Unmarshal([]byte(followeeStatsVal), &stats); err == nil {
			stats.Followers++
			if newVal, err := json.Marshal(stats); err == nil {
				r.client.HSet(ctx, followStatisticsKey, followeeStr, newVal)
			}
		}
	}

	return nil
}

// RemoveFollow 移除关注关系
func (r *RedisFollowCache) RemoveFollow(ctx context.Context, follower, followee int64) error {
	// 使用事务确保原子性
	pipe := r.client.Pipeline()
	followerStr := strconv.FormatInt(follower, 10)
	followeeStr := strconv.FormatInt(followee, 10)
	relationKey := fmt.Sprintf("%s%d", followRelationKeyPrefix, follower)

	// 1. 删除关注关系
	pipe.HDel(ctx, relationKey, followeeStr)

	// 2. 尝试更新关注者的统计数据
	followerStatsCmd := pipe.HGet(ctx, followStatisticsKey, followerStr)

	// 3. 尝试更新被关注者的统计数据
	followeeStatsCmd := pipe.HGet(ctx, followStatisticsKey, followeeStr)

	// 执行事务
	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}

	// 处理关注者统计数据
	followerStatsVal, err := followerStatsCmd.Result()
	if err == nil {
		var stats domain.FollowStatistics
		if err = json.Unmarshal([]byte(followerStatsVal), &stats); err == nil {
			stats.Followees--
			if stats.Followees < 0 {
				stats.Followees = 0
			}
			if newVal, err := json.Marshal(stats); err == nil {
				r.client.HSet(ctx, followStatisticsKey, followerStr, newVal)
			}
		}
	}

	// 处理被关注者统计数据
	followeeStatsVal, err := followeeStatsCmd.Result()
	if err == nil {
		var stats domain.FollowStatistics
		if err = json.Unmarshal([]byte(followeeStatsVal), &stats); err == nil {
			stats.Followers--
			if stats.Followers < 0 {
				stats.Followers = 0
			}
			if newVal, err := json.Marshal(stats); err == nil {
				r.client.HSet(ctx, followStatisticsKey, followeeStr, newVal)
			}
		}
	}

	return nil
}

// IsFollowing 判断某人是否关注了另一个人
func (r *RedisFollowCache) IsFollowing(ctx context.Context, follower, followee int64) (bool, error) {
	relationKey := fmt.Sprintf("%s%d", followRelationKeyPrefix, follower)
	followeeStr := strconv.FormatInt(followee, 10)

	// 使用HGet命令获取关注关系
	val, err := r.client.HGet(ctx, relationKey, followeeStr).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	status, err := strconv.Atoi(val)
	if err != nil {
		return false, err
	}
	return status == 1, nil
}
