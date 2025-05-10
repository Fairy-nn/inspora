package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Reward 打赏记录的数据库模型
type Reward struct {
	Id           int64  `gorm:"primaryKey,autoIncrement" bson:"id,omitempty"`
	Biz          string `gorm:"index:biz_biz_id"` // 业务类型，与BizId共同构成复合索引
	BizId        int64  `gorm:"index:biz_biz_id"` // 业务ID，与Biz共同构成复合索引
	BizName      string // 业务名称
	TargetUserId int64  `gorm:"index"` // 被打赏用户ID，建立单独索引
	Status       uint8  // 打赏状态：0-初始，1-成功，2-失败等
	UserId       int64  // 发起打赏的用户ID
	Amount       int64  // 打赏金额
	CreatedAt    int64  // 创建时间
	UpdatedAt    int64  // 更新时间
}

type RewardDAOInterface interface {
	Insert(ctx context.Context, r Reward) (int64, error)             // 插入打赏记录
	GetReward(ctx context.Context, rid int64) (Reward, error)        // 根据ID获取打赏记录
	UpdateStatus(ctx context.Context, rid int64, status uint8) error // 更新打赏状态
}

type RewardGORMDAO struct {
	db *gorm.DB
}

func NewRewardGORMDAO(db *gorm.DB) *RewardGORMDAO {
	return &RewardGORMDAO{
		db: db,
	}
}

// Insert 插入打赏记录
func (dao *RewardGORMDAO) Insert(ctx context.Context, r Reward) (int64, error) {
	now := time.Now().UnixMilli()
	r.CreatedAt = now
	r.UpdatedAt = now

	// 插入数据库
	result := dao.db.WithContext(ctx).Create(&r)
	return r.Id, result.Error
}

// GetReward 根据ID获取打赏记录
func (dao *RewardGORMDAO) GetReward(ctx context.Context, rid int64) (Reward, error) {
	var r Reward
	// 查询数据库
	result := dao.db.WithContext(ctx).Where("id = ?", rid).First(&r)
	return r, result.Error
}

// UpdateStatus 更新打赏状态
func (dao *RewardGORMDAO) UpdateStatus(ctx context.Context, rid int64, status uint8) error {
    return dao.db.WithContext(ctx).Model(&Reward{}).Where("id = ?", rid).Updates(
        map[string]any{
            "status":     status,           // 更新状态字段
            "updated_at": time.Now().UnixMilli(), // 更新更新时间
        },
    ).Error
}
