package repository

import (
	"context"
	"errors"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/repository/cache"
	"github.com/Fairy-nn/inspora/internal/repository/dao"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	ErrFollowSelf = errors.New("不能关注自己")
)

type FollowRepository interface {
	// Follow 关注用户
	Follow(ctx context.Context, follower, followee int64) error
	// CancelFollow 取消关注
	CancelFollow(ctx context.Context, follower, followee int64) error
	// IsFollowing 检查是否关注
	IsFollowing(ctx context.Context, follower, followee int64) (bool, error)
	// GetFolloweeList 获取关注列表
	GetFolloweeList(ctx context.Context, follower, offset, limit int64) ([]domain.FollowRelation, error)
	// GetFollowerList 获取粉丝列表
	GetFollowerList(ctx context.Context, followee, offset, limit int64) ([]domain.FollowRelation, error)
	// GetStatistics 获取统计数据
	GetStatistics(ctx context.Context, uid int64) (domain.FollowStatistics, error)
}

type CachedFollowRepository struct {
	dao   dao.FollowRelationDAO
	cache cache.FollowCache
}

func NewFollowRepository(dao dao.FollowRelationDAO, cache cache.FollowCache) FollowRepository {
	return &CachedFollowRepository{
		dao:   dao,
		cache: cache,
	}
}

// Follow 关注用户
func (r *CachedFollowRepository) Follow(ctx context.Context, follower, followee int64) error {
	if follower == followee {
		return ErrFollowSelf
	}
	// 检查数据库看是否已经关注
	rel, err := r.dao.FindFollowRelation(ctx, follower, followee)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// 更新或创建关注关系
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 创建新关注
		err = r.dao.CreateFollowRelation(ctx, dao.FollowRelation{
			Follower: follower,
			Followee: followee,
			Status:   dao.FollowRelationStatusActive,
		})
	} else if rel.Status == dao.FollowRelationStatusInactive {
		// 重新激活已取消的关注
		err = r.dao.UpdateStatus(ctx, followee, follower, dao.FollowRelationStatusActive)
	} else {
		// 已经是关注状态
		return nil
	}

	if err != nil {
		return err
	}

	// 更新缓存
	err = r.cache.AddFollow(ctx, follower, followee)
	if err != nil {
		// 缓存更新失败，但不影响主流程
		// TODO: 记录日志
	}

	// 更新统计数据
	r.updateStatistics(ctx, follower)
	r.updateStatistics(ctx, followee)

	return nil
}

// CancelFollow 取消关注
func (r *CachedFollowRepository) CancelFollow(ctx context.Context, follower, followee int64) error {
	// 检查数据库看是否已经关注
	rel, err := r.dao.FindFollowRelation(ctx, follower, followee)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	// 判断是否已经关注
	if rel.Status == dao.FollowRelationStatusActive {
		// 更新状态为未激活
		err = r.dao.UpdateStatus(ctx, followee, follower, dao.FollowRelationStatusInactive)
		if err != nil {
			return err
		}

		// 更新缓存
		err = r.cache.RemoveFollow(ctx, follower, followee)
		if err != nil {
			// 缓存更新失败，但不影响主流程
			// TODO: 记录日志
		}

		// 更新统计数据
		r.updateStatistics(ctx, follower)
		r.updateStatistics(ctx, followee)
	}
	return nil
}

// IsFollowing 检查是否关注
func (r *CachedFollowRepository) IsFollowing(ctx context.Context, follower, followee int64) (bool, error) {
	// 先检查缓存
	isFollowing, err := r.cache.IsFollowing(ctx, follower, followee)
	if err == nil {
		return isFollowing, nil
	}
	if err != redis.Nil {
		// 记录缓存错误但继续处理
		// TODO: 记录日志
	}

	// 如果缓存不存在，则查询数据库
	rel, err := r.dao.FindFollowRelation(ctx, follower, followee)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	return rel.Status == dao.FollowRelationStatusActive, nil
}

// GetFolloweeList 获取关注列表
func (r *CachedFollowRepository) GetFolloweeList(ctx context.Context, follower, offset, limit int64) ([]domain.FollowRelation, error) {
	// 直接查数据库
    daos, err := r.dao.FindFolloweeList(ctx, follower, offset, limit)
    if err != nil {
        return nil, err
    }

	// 转换为domain模型
    res := make([]domain.FollowRelation, 0, len(daos))
    for _, item := range daos {
        res = append(res, domain.FollowRelation{
            Follower: item.Follower,
            Followee: item.Followee,
        })
    }

    return res, nil
}

// GetFollowerList 获取粉丝列表
func (r *CachedFollowRepository) GetFollowerList(ctx context.Context, followee, offset, limit int64) ([]domain.FollowRelation, error) {
	// 直接查数据库
    daos, err := r.dao.FindFollowerList(ctx, followee, offset, limit)
    if err != nil {
        return nil, err
    }

	// 转换为domain模型
    res := make([]domain.FollowRelation, 0, len(daos))
    for _, item := range daos {
        res = append(res, domain.FollowRelation{
            Follower: item.Follower,
            Followee: item.Followee,
        })
    }

    return res, nil
}

// GetStatistics 获取统计数据
func (r *CachedFollowRepository) GetStatistics(ctx context.Context, uid int64) (domain.FollowStatistics, error) {
	// 先检查缓存
	stats, err := r.cache.GetStatistics(ctx, uid)
	if err == nil {
		return stats, nil
	}
	if err != redis.Nil {
		// 记录缓存错误但继续处理
		// TODO: 记录日志
	}

	// 如果缓存不存在，则查询数据库
	dbStats, err := r.dao.FindStatistics(ctx, uid)
    if err != nil {
        if !errors.Is(err, gorm.ErrRecordNotFound) {
            return domain.FollowStatistics{}, err
        }
        // 如果统计记录不存在，则重新计算
        return r.updateStatistics(ctx, uid)
    }

    // 转换为domain模型
    res := domain.FollowStatistics{
        Followers: dbStats.Followers,
        Followees: dbStats.Followees,
    }

    // 更新缓存
    r.cache.SetStatistics(ctx, uid, res)

    return res, nil
}

// updateStatistics 更新关注统计数据
func (r *CachedFollowRepository) updateStatistics(ctx context.Context, uid int64) (domain.FollowStatistics, error) {
	// 统计关注者数量
	followers, err := r.dao.CountFollowers(ctx, uid)
	if err != nil {
		return domain.FollowStatistics{}, err
	}

	// 统计关注数量
	followees, err := r.dao.CountFollowees(ctx, uid)
	if err != nil {
		return domain.FollowStatistics{}, err
	}

	// 更新数据库
	err = r.dao.UpsertStatistics(ctx, uid, followers, followees)
	if err != nil {
		return domain.FollowStatistics{}, err
	}

	// 统计数据
	stats := domain.FollowStatistics{
		Followers: followers,
		Followees: followees,
	}

	// 更新缓存
	r.cache.SetStatistics(ctx, uid, stats)

	// 返回统计数据
	return stats, nil
}
