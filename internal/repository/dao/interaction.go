package dao

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNotFound = gorm.ErrRecordNotFound

type InteractionDao struct {
	ID           int64  `gorm:"primaryKey;autoIncrement" json:"id"`       // 文章ID
	BizID        int64  `gorm:"uniqueIndex:idx_biz_id" json:"biz_id"`     // 业务ID
	Biz          string `gorm:"uniqueIndex:idx_biz_id;type:varchar(255)"` // 业务线
	ViewCount    int64  // 浏览量
	LikeCount    int64  // 点赞量
	CollectCount int64  // 收藏量
	Ctime        int64  // 创建时间
	Utime        int64  // 更新时间
}

// 用户点赞表，用户点赞的某个文章
type UserLikeBiz struct {
	ID int64 `gorm:"primaryKey,autoIncrement"`
	// 联合唯一索引
	Biz    string `gorm:"uniqueIndex:uid_biz_id_type;type:varchar(255)"`
	BizID  int64  `gorm:"uniqueIndex:uid_biz_id_type"`
	Uid    int64  `gorm:"uniqueIndex:uid_biz_id_type"`
	Ctime  int64
	Utime  int64
	Status uint8 // 交互状态
}

// 用户收藏表
type Collection struct {
	ID    int64  `gorm:"primaryKey,autoIncrement"`
	Name  string `gorm:"type:varchar(255)"`
	Uid   int64
	Ctime int64
	Utime int64
}

// 用户收藏表
type UserCollectionBiz struct {
	ID           int64  `gorm:"primaryKey,autoIncrement"`
	CollectionID int64  `gorm:"index"`
	Biz          string `gorm:"uniqueIndex:uid_biz_id_type;type:varchar(255)"`
	BizID        int64  `gorm:"uniqueIndex:uid_biz_id_type"`
	Uid          int64  `gorm:"uniqueIndex:uid_biz_id_type"`
	Ctime        int64
	Utime        int64
}

type InteractionDaoInterface interface {
	IncrViewCount(ctx context.Context, biz string, bizId int64) error
	InsertLikeInfo(ctx context.Context, biz string, bizId, uid int64) error
	GetLikeInfo(ctx context.Context, biz string, bizId, uid int64) (UserLikeBiz, error)
	DeleteLikeInfo(ctx context.Context, biz string, bizId, uid int64) error
	Get(ctx context.Context, biz string, bizId int64) (InteractionDao, error)
	InsertCollectionBiz(ctx context.Context, cb UserCollectionBiz) error
	GetCollectionInfo(ctx context.Context, biz string, bizId, uid int64) (UserCollectionBiz, error)
	DeleteCollectionInfo(ctx context.Context, biz string, bizId, uid int64) error
	BatchIncrReadCnt(ctx context.Context, biz []string, bizIds []int64) error
	GetByIds(ctx context.Context, biz string, ids []int64) ([]InteractionDao, error)
}

type GormInteractionDAO struct {
	db *gorm.DB
}

func NewGormInteractionDAO(db *gorm.DB) InteractionDaoInterface {
	return &GormInteractionDAO{
		db: db,
	}
}

// IncrViewCount 增加浏览量
func (i *GormInteractionDAO) IncrViewCount(ctx context.Context, biz string, bizId int64) error {
	// 更新时间
	now := time.Now().UnixMilli()
	// 先查询是否存在记录，如果不存在则插入一条新记录
	return i.db.WithContext(ctx).Clauses(clause.OnConflict{
		// 若记录已存在，则更新浏览量和更新时间
		DoUpdates: clause.Assignments(map[string]any{
			// 当发生冲突时，将 view_count 字段的值加 1
			"view_count": gorm.Expr("view_count + 1"),
			// 当发生冲突时，将 updated_at 字段的值更新为当前时间的毫秒级时间戳
			"utime": now,
		}),
	}).Create(&InteractionDao{ //没有主键冲突时，插入新记录
		BizID:     bizId,
		Biz:       biz,
		ViewCount: 1,
		Utime:     now,
		Ctime:     now,
	}).Error
}

// InsertLikeInfo 插入点赞信息
func (i *GormInteractionDAO) InsertLikeInfo(ctx context.Context, biz string, bizId, uid int64) error {
	now := time.Now().UnixMilli()
	return i.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.OnConflict{
			// 如果记录已存在，则更新点赞量和更新时间
			DoUpdates: clause.Assignments(map[string]any{
				"status": 1, // 1代表点赞
				"utime":  now,
			}),
		}).Create(&UserLikeBiz{
			Biz:    biz,
			BizID:  bizId,
			Uid:    uid,
			Utime:  now,
			Status: 1,
			Ctime:  now,
		}).Error
		if err != nil {
			return err
		}

		// 更新点赞量
		return tx.Clauses(clause.OnConflict{
			// 如果记录已存在，则更新点赞量和更新时间
			DoUpdates: clause.Assignments(map[string]any{
				"like_count": gorm.Expr("like_count + 1"),
				"utime":      now,
			}),
		}).Create(&InteractionDao{
			BizID:     bizId,
			Biz:       biz,
			LikeCount: 1,
			Utime:     now,
			Ctime:     now,
		}).Error
	})
}

// GetLikeInfo 获取点赞信息
func (i *GormInteractionDAO) GetLikeInfo(ctx context.Context, biz string, bizId, uid int64) (UserLikeBiz, error) {
	var likeInfo UserLikeBiz
	// 查询点赞信息
	err := i.db.WithContext(ctx).Where("biz = ? AND biz_id = ? AND uid = ?", biz, bizId, uid).First(&likeInfo).Error
	if err != nil {
		return UserLikeBiz{}, err
	}
	return likeInfo, nil
}

// DeleteLikeInfo 删除点赞信息
// 这里的“删除”是软删除，即只是修改状态而不是物理删除记录
func (i *GormInteractionDAO) DeleteLikeInfo(ctx context.Context, biz string, bizId, uid int64) error {
	now := time.Now().UnixMilli()
	// 这里使用事务来确保操作的原子性
	// 先将点赞状态设置为 0，表示取消点赞,然后更新点赞量
	return i.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&UserLikeBiz{}).Where("biz = ? AND biz_id = ? AND uid = ?", biz, bizId, uid).Updates(
			map[string]any{
				// 将点赞状态设置为 0，表示取消点赞。
				"status": 0,
				// 更新记录的更新时间为当前时间
				"updated_at": now,
			}).Error
		if err != nil {
			return err
		}

		// 更新点赞量
		return tx.Model(&InteractionDao{}).Where("biz = ? AND biz_id = ?", biz, bizId).Updates(
			map[string]any{
				"like_count": gorm.Expr("like_count - 1"),
				"utime":      time.Now().UnixMilli(),
			}).Error
	})
}

// Get 获取交互信息
func (i *GormInteractionDAO) Get(ctx context.Context, biz string, bizId int64) (InteractionDao, error) {
	var interaction InteractionDao
	err := i.db.WithContext(ctx).Where("biz = ? AND biz_id = ?", biz, bizId).First(&interaction).Error
	if err != nil {
		return InteractionDao{}, err
	}
	return interaction, nil
}

// InsertCollectionBiz 插入收藏信息
func (i *GormInteractionDAO) InsertCollectionBiz(ctx context.Context, cb UserCollectionBiz) error {
	now := time.Now().UnixMilli()
	cb.Utime = now
	cb.Ctime = now
	return i.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 插入收藏信息到数据库
		err := i.db.WithContext(ctx).Create(&cb).Error
		if err != nil {
			// 这里认为前端已经判断好了，不会重复插入
			return err
		}

		// 更新收藏量
		return tx.Clauses(clause.OnConflict{
			// 如果记录已存在，则更新收藏量和更新时间
			DoUpdates: clause.Assignments(map[string]any{
				"collect_count": gorm.Expr("collect_count + 1"),
				"utime":         now,
			}),
		}).Create(&InteractionDao{
			BizID:        cb.BizID,
			Biz:          cb.Biz,
			CollectCount: 1,
			Utime:        now,
			Ctime:        now,
		}).Error
	})
}

// GetCollectionInfo 获取收藏信息
func (i *GormInteractionDAO) GetCollectionInfo(ctx context.Context, biz string, bizId, uid int64) (UserCollectionBiz, error) {
	var collectionInfo UserCollectionBiz
	// 查询收藏信息
	err := i.db.WithContext(ctx).Where("biz = ? AND biz_id = ? AND uid = ?", biz, bizId, uid).First(&collectionInfo).Error
	// 打印生成的 SQL 语句
	// fmt.Println("SQL:", i.db.ToSQL(func(tx *gorm.DB) *gorm.DB {
	// 	return tx.Where("biz = ? AND biz_id = ? AND uid = ?", biz, bizId, uid).First(&collectionInfo)
	// }))
	if err != nil {
		return UserCollectionBiz{}, err
	}
	return collectionInfo, nil
}

// DeleteCollectionInfo 删除收藏信息
func (i *GormInteractionDAO) DeleteCollectionInfo(ctx context.Context, biz string, bizId, uid int64) error {
	return i.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除收藏信息
		err := tx.Where("biz = ? AND biz_id = ? AND uid = ?", biz, bizId, uid).Delete(&UserCollectionBiz{}).Error
		if err != nil {
			return err
		}

		// 更新收藏量
		return tx.Model(&InteractionDao{}).Where("biz = ? AND biz_id = ?", biz, bizId).Updates(
			map[string]any{
				"collect_count": gorm.Expr("collect_count - 1"),
				"utime":         time.Now().UnixMilli(),
			}).Error
	})
}

// 批量增加指定业务类型和业务 ID 对应的浏览计数
func (i *GormInteractionDAO) BatchIncrReadCnt(ctx context.Context, biz []string, bizIds []int64) error {
	return i.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		exDAO := NewGormInteractionDAO(tx)

		for i, b := range biz {
			// 调用 IncrViewCount 方法增加浏览量
			err := exDAO.IncrViewCount(ctx, b, bizIds[i])
			if err != nil {
				//return err
				fmt.Println("增加浏览量失败:", err)
				return err
			}
		}
		return nil
	})
}

// GetByIds 批量获取交互信息
func (dao *GormInteractionDAO) GetByIds(ctx context.Context, biz string, ids []int64) ([]InteractionDao, error) {
	var interactions []InteractionDao
	fmt.Println("GetByIds: ", biz, ids)
	err := dao.db.WithContext(ctx).Where("biz = ? AND biz_id IN ?", biz, ids).Find(&interactions).Error
	// 查找到的结果
	fmt.Printf("GetByIds: %v\n", interactions)
	if err != nil {
		return nil, err
	}
	return interactions, nil
}
