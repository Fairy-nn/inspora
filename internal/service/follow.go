package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/events/feed"
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
	repo     repository.FollowRepository
	feedProd feed.Producer
}

func NewFollowService(repo repository.FollowRepository, feedProd feed.Producer) FollowService {
	return &followService{
		repo:     repo,
		feedProd: feedProd,
	}
}

// Follow 关注用户
func (s *followService) Follow(ctx context.Context, follower, followee int64) error {
	//  return s.repo.Follow(ctx, follower, followee)
	err := s.repo.Follow(ctx, follower, followee)
	if err != nil {
		return err
	}

	// 发送用户关注事件
	if s.feedProd != nil {
		go func() {
			// 创建新的context，避免原context取消导致goroutine提前退出
			ctxTimeout, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			if err := s.feedProd.ProduceUserFollowedEvent(ctxTimeout, follower, followee); err != nil {
				fmt.Println("Failed to send user followed feed event:", err)
			}
		}()
	}

	return nil
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
