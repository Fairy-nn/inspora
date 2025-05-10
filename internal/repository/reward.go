package repository

import (
	"context"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/repository/cache"
	"github.com/Fairy-nn/inspora/internal/repository/dao"
)

type RewardRepositoryInterface interface {
	// 创建打赏记录
	CreateReward(ctx context.Context, reward domain.Reward) (int64, error)
	// 根据ID获取打赏记录
	GetReward(ctx context.Context, rid int64) (domain.Reward, error)
	// 获取缓存的二维码URL
	GetCachedCodeURL(ctx context.Context, r domain.Reward) (domain.CodeURL, error)
	// 缓存二维码URL
	CacheCodeURL(ctx context.Context, cu domain.CodeURL, r domain.Reward) error
	// 更新打赏状态
	UpdateStatus(ctx context.Context, rid int64, status domain.RewardStatus) error
}

type RewardRepository struct {
	dao   dao.RewardDAOInterface
	cache cache.RewardCacheInterface
}

func NewRewardRepository(dao dao.RewardDAOInterface, cache cache.RewardCacheInterface) RewardRepositoryInterface {
	return &RewardRepository{
		dao:   dao,
		cache: cache,
	}
}

// toDomain 将数据库模型转换为领域模型
func (r *RewardRepository) toDomain(rewarf dao.Reward) domain.Reward {
	return domain.Reward{
		ID:     rewarf.Id,
		UserID: rewarf.UserId,
		Target: domain.Target{
			Biz:     rewarf.Biz,
			BizId:   rewarf.BizId,
			BizName: rewarf.BizName,
		},
		Amt:    rewarf.Amount,
		Status: domain.RewardStatus(rewarf.Status),
	}
}

// toEntity 将领域模型转换为数据库模型
func (r *RewardRepository) toEntity(reward domain.Reward) dao.Reward {
	return dao.Reward{
		Id:      reward.ID,
		UserId:  reward.UserID,
		Biz:     reward.Target.Biz,
		BizId:   reward.Target.BizId,
		BizName: reward.Target.BizName,
		Status:  uint8(reward.Status),
		Amount:  reward.Amt,
	}
}

// CreateReward 创建打赏记录
func (r *RewardRepository) CreateReward(ctx context.Context, reward domain.Reward) (int64, error) {
	entity := r.toEntity(reward)
	// 借用dao的Insert方法
	return r.dao.Insert(ctx, entity)
}

// GetReward 根据ID获取打赏记录
func (r *RewardRepository) GetReward(ctx context.Context, rid int64) (domain.Reward, error) {
	// 调用dao的GetReward方法
	entity, err := r.dao.GetReward(ctx, rid)
	if err != nil {
		return domain.Reward{}, err
	}
	return r.toDomain(entity), nil
}

// GetCachedCodeURL 获取缓存的二维码URL
func (r *RewardRepository) GetCachedCodeURL(ctx context.Context, rr domain.Reward) (domain.CodeURL, error) {
	return r.cache.GetCachedCodeURL(ctx, rr)
}

// CacheCodeURL 缓存二维码URL
func (r *RewardRepository) CacheCodeURL(ctx context.Context, cu domain.CodeURL, rr domain.Reward) error {
	return r.cache.CachedCodeURL(ctx, cu, rr)
}

// UpdateStatus 更新打赏状态
func (r *RewardRepository) UpdateStatus(ctx context.Context, rid int64, status domain.RewardStatus) error {
	return r.dao.UpdateStatus(ctx, rid, uint8(status))
}
