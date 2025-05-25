package dao

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	"gorm.io/gorm"
)

// FeedEvent 是 Feed 事件的数据库模型
type FeedEvent struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	UserID    int64     `gorm:"column:user_id;type:int(11);not null;index"`
	EventType string    `gorm:"column:event_type;type:varchar(50);not null;index"`
	Content   string    `gorm:"column:content;type:text;not null"` // JSON 格式存储
	Ctime     time.Time `gorm:"column:created_at;not null;index"`
}

type FeedDAOInterface interface {
	// SaveFeedEvent 将 Feed 事件保存到数据库
	SaveFeedEvent(ctx context.Context, event domain.FeedEvent) (int64, error)
	// GetFeedEventByID 根据 ID 获取 Feed 事件
	GetFeedEventByID(ctx context.Context, id int64) (FeedEvent, error)
	// GetFeedEventsSince 获取指定时间之后的所有 Feed 事件
	GetFeedEventsSince(ctx context.Context, since time.Time, offset, limit int) ([]FeedEvent, error)
	// GetFeedEventsForUser 获取指定用户的 Feed 事件
	GetFeedEventsForUser(ctx context.Context, userID int64, offset, limit int) ([]FeedEvent, error)
}

// GORMFeedDAO 是基于 GORM 的 Feed DAO 实现
type GORMFeedDAO struct {
	db *gorm.DB
}

// NewGORMFeedDAO 创建一个新的 GORM Feed DAO
func NewGORMFeedDAO(db *gorm.DB) FeedDAOInterface {
	return &GORMFeedDAO{
		db: db,
	}
}

// SaveFeedEvent 将 Feed 事件保存到数据库
func (g *GORMFeedDAO) SaveFeedEvent(ctx context.Context, event domain.FeedEvent) (int64, error) {
	// 将领域模型转换为数据库模型
	contentJSON, err := json.Marshal(event.Content)
	if err != nil {
		return 0, err
	}

	feedEvent := FeedEvent{
		UserID:    event.UserID,
		EventType: event.EventType,
		Content:   string(contentJSON),
		Ctime:     event.Ctime,
	}

	// 保存到数据库
	result := g.db.WithContext(ctx).Create(&feedEvent)
	if result.Error != nil {
		return 0, result.Error
	}

	return feedEvent.ID, nil
}

// GetFeedEventByID 根据 ID 获取 Feed 事件
func (g *GORMFeedDAO) GetFeedEventByID(ctx context.Context, id int64) (FeedEvent, error) {
	var feedEvent FeedEvent
	result := g.db.WithContext(ctx).First(&feedEvent, id)
	return feedEvent, result.Error
}

// GetFeedEventsSince 获取指定时间之后的所有 Feed 事件
func (g *GORMFeedDAO) GetFeedEventsSince(ctx context.Context, since time.Time, offset, limit int) ([]FeedEvent, error) {
	var events []FeedEvent
	result := g.db.WithContext(ctx).
		Where("ctime >= ?", since).
		Order("ctime DESC").
		Offset(offset).
		Limit(limit).
		Find(&events)

	return events, result.Error
}

// GetFeedEventsForUser 获取指定用户的 Feed 事件
func (g *GORMFeedDAO) GetFeedEventsForUser(ctx context.Context, userID int64, offset, limit int) ([]FeedEvent, error) {
	var events []FeedEvent
	result := g.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("ctime DESC").
		Offset(offset).
		Limit(limit).
		Find(&events)

	return events, result.Error
}
