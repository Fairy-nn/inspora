package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/events/feed"
	"github.com/Fairy-nn/inspora/internal/repository"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

type InteractionServiceInterface interface {
	IncrViewCount(ctx context.Context, biz string, id int64) error
	Like(ctx context.Context, biz string, bizId int64, uid int64) error
	CancelLike(ctx context.Context, biz string, bizId int64, uid int64) error
	// cid是收藏夹的id
	Collect(ctx context.Context, biz string, bizId int64, cid, uid int64) error
	CancelCollect(ctx context.Context, biz string, bizId int64, cid, uid int64) error
	Get(ctx context.Context, biz string, bizId, uid int64) (domain.Interaction, error)
	GetByIds(ctx context.Context, biz string, ids []int64) (map[int64]domain.Interaction, error)
}

type InteractionService struct {
	repo       repository.InteractionRepositoryInterface
	feedProd   feed.Producer
	articleSvc ArticleServiceInterface
}

// 创建一个新的交互服务实例
func NewInteractionService(repo repository.InteractionRepositoryInterface, feedProd feed.Producer, articleSvc ArticleServiceInterface) InteractionServiceInterface {
	return &InteractionService{
		repo:       repo,
		feedProd:   feedProd,
		articleSvc: articleSvc,
	}
}

// IncrViewCount 增加浏览量
func (i *InteractionService) IncrViewCount(ctx context.Context, biz string, id int64) error {
	return i.repo.IncrViewCount(ctx, biz, id)
}

// Like 增加点赞量
func (i *InteractionService) Like(ctx context.Context, biz string, bizId int64, uid int64) error {
	err := i.repo.IncrLikeCount(ctx, biz, bizId, uid)
	if err != nil {
		return err
	}
	// 发送用户点赞事件
	if biz == "article" && i.feedProd != nil {
		// 获取文章作者ID
		var authorID int64

		// 从文章服务获取文章信息
		article, err := i.articleSvc.FindPublicArticleById(ctx, bizId, uid)
		if err != nil {
			// 仅记录错误日志，不影响主流程
			fmt.Printf("获取文章信息失败: %v\n", err)
		} else {
			authorID = article.Author.ID
		}

		// 异步发送Feed事件，避免阻塞主流程
		go func(aid int64) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := i.feedProd.ProduceArticleLikedEvent(ctx, uid, bizId, aid); err != nil {
				// 记录错误日志，但不影响主流程
				fmt.Printf("发送点赞Feed事件失败: %v\n", err)
			}
		}(authorID)
	}
	return nil
}

// CancelLike 减少点赞量
func (i *InteractionService) CancelLike(ctx context.Context, biz string, bizId int64, uid int64) error {

	return i.repo.DecrLikeCount(ctx, biz, bizId, uid)
}

// Collect 增加收藏量
func (i *InteractionService) Collect(ctx context.Context, biz string, bizId int64, cid, uid int64) error {
	err := i.repo.AddCollectionItem(ctx, biz, bizId, cid, uid)
	if err != nil {
		fmt.Printf("添加收藏失败: %v\n", err)
	}
	// 发送用户收藏事件
	// 发送Feed事件（仅对文章收藏发送）
	if biz == "article" && i.feedProd != nil {
		// 获取文章作者ID
		var authorID int64

		// 从文章服务获取文章信息
		article, err := i.articleSvc.FindPublicArticleById(ctx, bizId, uid)
		if err != nil {
			// 仅记录错误日志，不影响主流程
			fmt.Printf("获取文章信息失败: %v\n", err)
		} else {
			authorID = article.Author.ID
		}

		// 异步发送Feed事件，避免阻塞主流程
		go func(aid int64) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := i.feedProd.ProduceArticleCollectedEvent(ctx, uid, bizId, aid); err != nil {
				// 记录错误日志，但不影响主流程
				fmt.Printf("发送收藏Feed事件失败: %v\n", err)
			}
		}(authorID)
	}

	return nil
}

// CancelCollect 减少收藏量
func (i *InteractionService) CancelCollect(ctx context.Context, biz string, bizId int64, cid, uid int64) error {
	return i.repo.RemoveCollectionItem(ctx, biz, bizId, cid, uid)
}

// Get 获取交互信息
func (i *InteractionService) Get(ctx context.Context, biz string, bizId, uid int64) (domain.Interaction, error) {
	var (
		eg          errgroup.Group
		interaction domain.Interaction // 用于存储从仓库层获取的交互信息
		liked       bool               // 用于存储点赞状态
		collected   bool               // 用于存储收藏状态
		getIntErr   error              // 用于存储获取交互信息时的错误
	)

	// 并发获取交互信息
	eg.Go(func() error {
		interaction, getIntErr = i.repo.Get(ctx, biz, bizId)
		return getIntErr
	})

	//  获得点赞状态
	eg.Go(func() error {
		var err error
		liked, err = i.repo.Liked(ctx, biz, bizId, uid)
		return err
	})

	// 并发获取用户的收藏状态
	eg.Go(func() error {
		var err error
		collected, err = i.repo.Collected(ctx, biz, bizId, uid)
		return err
	})

	// 等待所有的 goroutine 完成
	err := eg.Wait()
	if err != nil {
		// 检查整体错误是否为记录未找到的错误
		if errors.Is(getIntErr, gorm.ErrRecordNotFound) {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 如果是记录未找到的错误，将获取到的点赞和收藏状态赋值给交互信息对象
				interaction.Liked = liked
				interaction.Collected = collected
				return interaction, nil
			}
		}

		return interaction, err
	}

	interaction.Liked = liked
	interaction.Collected = collected

	return interaction, nil
}

// Liked 判断是否点赞
func (i *InteractionService) Liked(ctx context.Context, biz string, bizId, uid int64) (bool, error) {
	return i.repo.Liked(ctx, biz, bizId, uid)
}

// Collected 判断是否收藏
func (i *InteractionService) Collected(ctx context.Context, biz string, bizId, uid int64) (bool, error) {
	return i.repo.Collected(ctx, biz, bizId, uid)
}

// GetByIds 批量获取交互信息
func (i *InteractionService) GetByIds(ctx context.Context, biz string, ids []int64) (map[int64]domain.Interaction, error) {
	return i.repo.GetByIds(ctx, biz, ids)
}
