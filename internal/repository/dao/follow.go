package dao

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// 关注关系表
type FollowRelation struct {
	ID int64 `gorm:"primaryKey,autoIncrement,column:id"`

	// 被关注的人
	Follower int64 `gorm:"type:int(11);not null;uniqueIndex:follower_followee"`
	Followee int64 `gorm:"type:int(11);not null;uniqueIndex:follower_followee"`

	Status uint8

	CreatedAt int64
	UpdatedAt int64
}

// 关注统计表
type FollowStatistics struct {
	ID  int64 `gorm:"primaryKey,autoIncrement,column:id"`
	Uid int64 `gorm:"unique"`
	// 粉丝数量
	Followers int64
	// 关注数量
	Followees int64

	UpdatedAt int64
	CreatedAt int64
}

const (
	FollowRelationStatusUnknown uint8 = iota
	FollowRelationStatusActive
	FollowRelationStatusInactive
)

type FollowRelationDAO interface {
	// FindFolloweeList 获取某人的关注列表
	FindFolloweeList(ctx context.Context, follower, offset, limit int64) ([]FollowRelation, error)
	// FindFollowerList 获取某人的粉丝列表
	FindFollowerList(ctx context.Context, followee, offset, limit int64) ([]FollowRelation, error)
	// FindFollowRelation 查询关注关系
	FindFollowRelation(ctx context.Context, follower int64, followee int64) (FollowRelation, error)
	// CreateFollowRelation 创建关注关系
	CreateFollowRelation(ctx context.Context, fr FollowRelation) error
	// UpdateStatus 更新关注状态
	UpdateStatus(ctx context.Context, followee int64, follower int64, status uint8) error
	// CountFollowers 统计粉丝数量
	CountFollowers(ctx context.Context, uid int64) (int64, error)
	// CountFollowees 统计关注数量
	CountFollowees(ctx context.Context, uid int64) (int64, error)
	// UpsertStatistics 更新或插入统计数据
	UpsertStatistics(ctx context.Context, uid int64, followers int64, followees int64) error
	// FindStatistics 查询统计数据
	FindStatistics(ctx context.Context, uid int64) (FollowStatistics, error)
}

type GORMFollowRelationDAO struct {
	db *gorm.DB
}

func NewFollowRelationDAO(db *gorm.DB) FollowRelationDAO {
	return &GORMFollowRelationDAO{
		db: db,
	}
}

// FindFolloweeList 获取某人的关注列表
func (dao *GORMFollowRelationDAO) FindFolloweeList(ctx context.Context, follower, offset, limit int64) ([]FollowRelation, error) {
	var res []FollowRelation
	err := dao.db.WithContext(ctx).Where("follower = ? AND status = ?", follower, FollowRelationStatusActive).
		Offset(int(offset)).Limit(int(limit)).Order("id DESC").Find(&res).Error
	return res, err
}

// FindFollowerList 获取某人的粉丝列表
func (dao *GORMFollowRelationDAO) FindFollowerList(ctx context.Context, followee, offset, limit int64) ([]FollowRelation, error) {
	var res []FollowRelation
	err := dao.db.WithContext(ctx).Where("followee = ? AND status = ?", followee, FollowRelationStatusActive).
		Offset(int(offset)).Limit(int(limit)).Order("id DESC").Find(&res).Error
	return res, err
}

// FindFollowRelation 查询关注关系
func (dao *GORMFollowRelationDAO) FindFollowRelation(ctx context.Context, follower int64, followee int64) (FollowRelation, error) {
	var res FollowRelation
	err := dao.db.WithContext(ctx).Where("follower = ? AND followee = ?", follower, followee).First(&res).Error
	return res, err
}

// CreateFollowRelation 创建关注关系
func (dao *GORMFollowRelationDAO) CreateFollowRelation(ctx context.Context, fr FollowRelation) error {
	now := time.Now().Unix()
	fr.CreatedAt = now
	fr.UpdatedAt = now
	return dao.db.WithContext(ctx).Create(&fr).Error
}

// UpdateStatus 更新关注状态
func (dao *GORMFollowRelationDAO) UpdateStatus(ctx context.Context, followee int64, follower int64, status uint8) error {
	return dao.db.WithContext(ctx).Model(&FollowRelation{}).
		Where("follower = ? AND followee = ?", follower, followee).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now().Unix(),
		}).Error
}

// CountFollowers 统计粉丝数量
func (dao *GORMFollowRelationDAO) CountFollowers(ctx context.Context, uid int64) (int64, error) {
	var count int64
	err := dao.db.WithContext(ctx).Model(&FollowRelation{}).
		Where("followee = ? AND status = ?", uid, FollowRelationStatusActive).
		Count(&count).Error
	return count, err
}

// CountFollowees 统计关注数量
func (dao *GORMFollowRelationDAO) CountFollowees(ctx context.Context, uid int64) (int64, error) {
	var count int64
	err := dao.db.WithContext(ctx).Model(&FollowRelation{}).
		Where("follower = ? AND status = ?", uid, FollowRelationStatusActive).
		Count(&count).Error
	return count, err
}

// UpsertStatistics 更新或插入统计数据
// 这里使用了事务来保证原子性,，防止在高并发情况下出现数据不一致。
// 这个函数主要用于维护每个用户的关注统计信息，
// 包括他们关注了多少人（followees）以及有多少粉丝（followers），
// 这些数据会用于用户个人资料页面的展示和数据分析。
func (dao *GORMFollowRelationDAO) UpsertStatistics(ctx context.Context, uid int64, followers int64, followees int64) error {
	now := time.Now().UnixMilli()

	return dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var stats FollowStatistics
		err := tx.Where("uid = ?", uid).First(&stats).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Record does not exist, create a new record
				stats = FollowStatistics{
					Uid:       uid,
					Followers: followers,
					Followees: followees,
					CreatedAt: now,
					UpdatedAt: now,
				}
				return tx.Create(&stats).Error
			}
			return err
		}

		// Record exists, update
		stats.Followers = followers
		stats.Followees = followees
		stats.UpdatedAt = now
		return tx.Save(&stats).Error
	})
}

// FindStatistics 查询统计数据
func (dao *GORMFollowRelationDAO) FindStatistics(ctx context.Context, uid int64) (FollowStatistics, error) {
	var res FollowStatistics
	err := dao.db.WithContext(ctx).Where("uid = ?", uid).First(&res).Error
	return res, err
}
