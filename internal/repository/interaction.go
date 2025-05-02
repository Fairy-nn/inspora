package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/repository/cache"
	"github.com/Fairy-nn/inspora/internal/repository/dao"
)

type InteractionRepositoryInterface interface {
	IncrViewCount(ctx context.Context, biz string, bizId int64) error
	IncrLikeCount(ctx context.Context, biz string, bizId, uid int64) error
	DecrLikeCount(ctx context.Context, biz string, bizId, uid int64) error
	AddCollectionItem(ctx context.Context, biz string, bizId, cid, uid int64) error
	Get(ctx context.Context, biz string, bizId int64) (domain.Interaction, error)
	Liked(ctx context.Context, biz string, bizId, uid int64) (bool, error)
	Collected(ctx context.Context, biz string, bizId, uid int64) (bool, error)
	RemoveCollectionItem(ctx context.Context, biz string, bizId, cid, uid int64) error
}

type InteractionRepository struct {
	dao   dao.InteractionDaoInterface
	cache cache.InteractionCacheInterface
}

func NewInteractionRepository(dao dao.InteractionDaoInterface, cache cache.InteractionCacheInterface) InteractionRepositoryInterface {
	return &InteractionRepository{
		dao:   dao,
		cache: cache,
	}
}

// IncrViewCount 增加浏览量
func (i *InteractionRepository) IncrViewCount(ctx context.Context, biz string, bizId int64) error {
	// 先增加数据库中的浏览量
	err := i.dao.IncrViewCount(ctx, biz, bizId)
	if err != nil {
		return err
	}

	// 然后增加缓存中的浏览量
	return i.cache.IncrViewCntIfPresent(ctx, biz, bizId)
}

// IncrLikeCount 增加点赞量
func (i *InteractionRepository) IncrLikeCount(ctx context.Context, biz string, bizId, uid int64) error {
	// 先增加数据库中的点赞量
	err := i.dao.InsertLikeInfo(ctx, biz, bizId, uid)
	if err != nil {
		return err
	}

	// 然后增加缓存中的点赞量
	return i.cache.IncrLikeCntIfPresent(ctx, biz, bizId)
}

// DecrLikeCount 减少点赞量
func (i *InteractionRepository) DecrLikeCount(ctx context.Context, biz string, bizId, uid int64) error {
	// 先删除数据库中的点赞信息
	err := i.dao.DeleteLikeInfo(ctx, biz, bizId, uid)
	if err != nil {
		return err
	}

	// 然后减少缓存中的点赞量
	return i.cache.DecrLikeCntIfPresent(ctx, biz, bizId)
}

// AddCollectionItem 添加收藏项
func (i *InteractionRepository) AddCollectionItem(ctx context.Context, biz string, bizId, cid, uid int64) error {
	// 先插入数据库中的收藏信息
	err := i.dao.InsertCollectionBiz(ctx, dao.UserCollectionBiz{
		Biz:          biz,
		BizID:        bizId,
		CollectionID: cid,
		Uid:          uid,
		Utime:        time.Now().UnixMilli(),
	})
	if err != nil {
		return err
	}

	// 然后增加缓存中的收藏量
	return i.cache.IncrCollectCntIfPresent(ctx, biz, bizId)
}

// Get 获取交互信息
func (i *InteractionRepository) Get(ctx context.Context, biz string, bizId int64) (domain.Interaction, error) {
	// 先从缓存中获取交互信息
	interaction, err := i.cache.Get(ctx, biz, bizId)
	if err == nil {
		return interaction, nil
	}

	// 如果缓存中没有，则从数据库中获取
	interEntity, err := i.dao.Get(ctx, biz, bizId)
	if err != nil {
		return domain.Interaction{}, err
	}

	interaction = i.toDomain(interEntity)

	// 将获取到的交互信息存入缓存中
	go func() {
		err := i.cache.Set(ctx, biz, bizId, interaction)
		if err != nil {
			fmt.Println("缓存交互信息失败:", err)
		}
	}()
	return interaction, nil
}

// Liked 判断是否点赞
func (i *InteractionRepository) Liked(ctx context.Context, biz string, bizId, uid int64) (bool, error) {
	_, err := i.dao.GetLikeInfo(ctx, biz, bizId, uid)
	switch err {
	case nil:
		return true, nil
	case dao.ErrNotFound:
		return false, nil
	default:
		return false, err
	}
}

// Collected 判断是否收藏
func (i *InteractionRepository) Collected(ctx context.Context, biz string, bizId, uid int64) (bool, error) {
	_, err := i.dao.GetCollectionInfo(ctx, biz, bizId, uid)
	switch err {
	case nil:
		return true, nil
	case dao.ErrNotFound:
		return false, nil
	default:
		return false, err
	}
}

// RemoveCollectionItem 删除收藏项
func (i *InteractionRepository) RemoveCollectionItem(ctx context.Context, biz string, bizId, cid, uid int64) error {
	// 先删除数据库中的收藏信息
	err := i.dao.DeleteCollectionInfo(ctx, biz, bizId, uid)
	if err != nil {
		return err
	}

	// 然后减少缓存中的收藏量
	return i.cache.DecrCollectCntIfPresent(ctx, biz, bizId)
}

// toDomain 将数据库实体转换为领域模型
func (i *InteractionRepository) toDomain(interEntity dao.InteractionDao) domain.Interaction {
	return domain.Interaction{
		ViewCnt:    interEntity.ViewCount,
		LikeCnt:    interEntity.LikeCount,
		CollectCnt: interEntity.CollectCount,
	}
}
