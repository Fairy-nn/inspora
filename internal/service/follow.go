package service

import (
	"context"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/repository"
)

type FollowService interface {
	// Follow 关注用户
	Follow(ctx context.Context, follower, followee int64) error
	// CancelFollow 取消关注
	CancelFollow(ctx context.Context, follower, followee int64) error
	// GetFollowInfo 获取关注信息
	GetFollowInfo(ctx context.Context, follower, followee int64) (bool, error)
	// GetFolloweeList 获取关注列表
	GetFolloweeList(ctx context.Context, follower, offset, limit int64) ([]domain.FollowRelation, error)
	// GetFollowerList 获取粉丝列表
	GetFollowerList(ctx context.Context, followee, offset, limit int64) ([]domain.FollowRelation, error)
	// GetFollowStatistics 获取统计数据
	GetFollowStatistics(ctx context.Context, uid int64) (domain.FollowStatistics, error)
}

type followService struct {
	repo repository.FollowRepository
}

func NewFollowService(repo repository.FollowRepository) FollowService {
	return &followService{
		repo: repo,
	}
}

// Follow 关注用户
func (s *followService) Follow(ctx context.Context, follower, followee int64) error {
	 return s.repo.Follow(ctx, follower, followee)
}

// CancelFollow 取消关注
func (s *followService) CancelFollow(ctx context.Context, follower, followee int64) error {
	return s.repo.CancelFollow(ctx, follower, followee)
}

// GetFollowInfo 获取关注信息
func (s *followService) GetFollowInfo(ctx context.Context, follower, followee int64) (bool, error) {
 return s.repo.IsFollowing(ctx, follower, followee)
}

// GetFolloweeList 获取关注列表
func (s *followService) GetFolloweeList(ctx context.Context, follower, offset, limit int64) ([]domain.FollowRelation, error) {
	return s.repo.GetFolloweeList(ctx, follower, offset, limit)
}

// GetFollowerList 获取粉丝列表
func (s *followService) GetFollowerList(ctx context.Context, followee, offset, limit int64) ([]domain.FollowRelation, error) {
return s.repo.GetFollowerList(ctx, followee, offset, limit)
}

// GetFollowStatistics 获取统计数据
func (s *followService) GetFollowStatistics(ctx context.Context, uid int64) (domain.FollowStatistics, error) {
 return s.repo.GetStatistics(ctx, uid)
}