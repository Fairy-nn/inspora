package dao

import (
	"context"
	"database/sql"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	"gorm.io/gorm"
)

// Payment 表示数据库中的支付记录结构
type Payment struct {
	Id          int64 `gorm:"primaryKey,autoIncrement" bson:"id,omitempty"`
	Amt         int64
	Currency    string
	Description string         `gorm:"description"`
	BizTradeNO  string         `gorm:"column:biz_trade_no;type:varchar(256);unique"`
	TxnID       sql.NullString `gorm:"column:txn_id;type:varchar(128);unique"` // Txn ID from WeChat Pay
	Status      uint8
	UpdatedAt   int64 `gorm:"index"`
	CreatedAt   int64
}

type PaymentDAOInterface interface {
	Insert(ctx context.Context, payment Payment) error
	UpdatedTxnIDAndStatus(ctx context.Context, buzTradeNO string, txnID string, status domain.PaymentStatus) error
	FindExpiredPayment(ctx context.Context, offset int, limit int, t time.Time) ([]Payment, error)
	GetPayment(ctx context.Context, bizTradeNO string) (Payment, error)
}

type PaymentGORMDAO struct {
	db *gorm.DB
}

func NewPaymentGORMDAO(db *gorm.DB) PaymentDAOInterface {
	return &PaymentGORMDAO{
		db: db,
	}
}

// Insert 插入支付记录
func (dao *PaymentGORMDAO) Insert(ctx context.Context, payment Payment) error {
	now := time.Now().UnixMilli()
	payment.CreatedAt = now
	payment.UpdatedAt = now
	// 使用GORM创建记录
	if err := dao.db.WithContext(ctx).Create(&payment).Error; err != nil {
		return err
	}
	return nil
}

// UpdatedTxnIDAndStatus 更新支付记录的交易ID和状态
func (dao *PaymentGORMDAO) UpdatedTxnIDAndStatus(ctx context.Context, bizTradeNO string, txnID string, status domain.PaymentStatus) error {
	return dao.db.WithContext(ctx).Model(&Payment{}).Where("biz_trade_no = ?", bizTradeNO).Updates(map[string]any{
		"txn_id":     txnID,                  // 更新交易ID
		"status":     status.AsUint8(),       // 更新支付状态
		"updated_at": time.Now().UnixMilli(), // 更新更新时间
	}).Error
}

// FindExpiredPayment 实现查询过期支付记录的方法
func (dao *PaymentGORMDAO) FindExpiredPayment(ctx context.Context, offset int, limit int, t time.Time) ([]Payment, error) {
	var payments []Payment
	err := dao.db.WithContext(ctx).Where("status = ? AND updated_at < ?", domain.PaymentStatusInit, t.UnixMilli()).Offset(offset).Limit(limit).Find(&payments).Error
	return payments, err
}

// GetPayment 根据业务交易号获取支付记录
func (dao *PaymentGORMDAO) GetPayment(ctx context.Context, bizTradeNO string) (Payment, error) {
	var payment Payment
	// 构建查询：根据业务交易号查找记录
	err := dao.db.WithContext(ctx).Where("biz_trade_no = ?", bizTradeNO).First(&payment).Error
	if err != nil {
		return Payment{}, err
	}
	return payment, nil
}
