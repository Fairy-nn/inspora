package repository

import (
	"context"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/repository/dao"
)

type PaymentRepositoryInterface interface {
	// 向数据库中添加一条支付记录，将支付信息存储到数据库中
	AddPayment(ctx context.Context, payment domain.Payment) error
	// 根据交易号更新数据库中的支付记录，主要更新交易ID和支付状态
	UpdatePayment(ctx context.Context, payment domain.Payment) error
	// 查询过期的支付记录
	FindExpiredPayments(ctx context.Context, offset int, limit int, t time.Time) ([]domain.Payment, error)
	// 根据业务交易号获取支付记录
	GetPayment(ctx context.Context, bizTradeNO string) (domain.Payment, error)
}

type PaymentRepository struct {
	dao dao.PaymentDAOInterface // 支付数据访问对象接口，用于执行底层的数据库操作
}

func NewPaymentRepository(dao dao.PaymentDAOInterface) PaymentRepositoryInterface {
	return &PaymentRepository{
		dao: dao,
	}
}

// toDomain 将数据库中的支付记录转换为领域模型
func (r *PaymentRepository) toDomain(payment dao.Payment) domain.Payment {
	return domain.Payment{
		Amt: domain.Amount{
			Currency: payment.Currency, // 货币类型
			Total:    payment.Amt, // 支付金额
		},
		BizTradeNo:  payment.BizTradeNO, // 业务交易号
		Description: payment.Description, // 支付描述
		Status:      domain.PaymentStatus(payment.Status), // 支付状态
		TxnID:       payment.TxnID.String, // 交易ID
	}
}

// toEntity 将领域模型转换为数据库中的支付记录
func (r *PaymentRepository) toEntity(payment domain.Payment) dao.Payment {
	return dao.Payment{
		Amt:         payment.Amt.Total,
		Currency:    payment.Amt.Currency,
		Description: payment.Description,
		BizTradeNO:  payment.BizTradeNo,
		Status:      payment.Status.AsUint8(),
	}
}

// AddPayment 向数据库中添加一条支付记录
func (r *PaymentRepository) AddPayment(ctx context.Context, payment domain.Payment) error {
	paymentEntity := r.toEntity(payment)
	return r.dao.Insert(ctx, paymentEntity)
}

// UpdatePayment 根据交易号更新数据库中的支付记录，主要更新交易ID和支付状态
func (r *PaymentRepository) UpdatePayment(ctx context.Context, payment domain.Payment) error {
	return r.dao.UpdatedTxnIDAndStatus(ctx, payment.BizTradeNo, payment.TxnID, payment.Status)
}

// FindExpiredPayments 查询过期的支付记录
func (r *PaymentRepository) FindExpiredPayments(ctx context.Context, offset int, limit int, t time.Time) ([]domain.Payment, error) {
	payments, err := r.dao.FindExpiredPayment(ctx, offset, limit, t)
	if err != nil {
		return nil, err
	}
	domainPayments := make([]domain.Payment, len(payments))
	for i, payment := range payments {
		domainPayments[i] = r.toDomain(payment)
	}
	return domainPayments, nil
}

// GetPayment 根据业务交易号获取支付记录
func (r *PaymentRepository) GetPayment(ctx context.Context, bizTradeNO string) (domain.Payment, error) {
	payment, err := r.dao.GetPayment(ctx, bizTradeNO)
	return r.toDomain(payment), err
}
