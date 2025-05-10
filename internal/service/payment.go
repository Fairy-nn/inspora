package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	events "github.com/Fairy-nn/inspora/internal/events/article"
	"github.com/Fairy-nn/inspora/internal/repository"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
)

type PaymentServiceInterface interface {
	// Prepay 根据提供的支付信息生成预支付URL
	Prepay(ctx context.Context, payment domain.Payment) (string, error)
	// HandleCallback 处理微信支付回调
	HandleCallback(ctx context.Context, txn *payments.Transaction) error
	// SyncWechatInfo 同步微信支付信息-对账功能
	SyncWechatInfo(ctx context.Context, BizTradeNo string) error
	// FindExpiredPayment 查询过期的支付记录
	FindExpiredPayment(ctx context.Context, offset int, limit int, t time.Time) ([]domain.Payment, error)
	// GetPayment 根据业务交易号获取支付记录
	GetPayment(ctx context.Context, bizTradeNO string) (domain.Payment, error)
}

type NativePaymentService struct {
	svc       *native.NativeApiService              // 微信原生支付API客户端
	appID     string                                // 微信应用ID
	mchid     string                                // 微信商户ID，用于标识商户身份
	notifyURL string                                // 支付结果通知URL
	repo      repository.PaymentRepositoryInterface // 支付数据仓储接口
	// status 映射微信支付状态到本地定义的支付状态
	status   map[string]domain.PaymentStatus
	producer events.Producer // 事件生产者，用于发布支付状态变更事件
}

func NewNativePaymentService(svc *native.NativeApiService, appID string, mchid string, repo repository.PaymentRepositoryInterface) *NativePaymentService {
	return &NativePaymentService{
		svc:       svc,
		appID:     appID,
		mchid:     mchid,
		notifyURL: "https://wechat.jetstudio.tech/pay/callback",
		repo:      repo,
		status: map[string]domain.PaymentStatus{
			"SUCCESS":    domain.PaymentStatusSuccess, // 支付成功
			"PAYERROR":   domain.PaymentStatusFailed,  // 支付失败
			"NOTPAY":     domain.PaymentStatusInit,    // 未支付
			"USERPAYING": domain.PaymentStatusInit,    // 用户支付中
			"CLOSED":     domain.PaymentStatusFailed,  // 订单已关闭
			"REVOKED":    domain.PaymentStatusFailed,  // 订单已撤销
			"REFUND":     domain.PaymentStatusRefund,  // 订单已退款
		},
	}
}

// Prepay 实现预支付功能，生成微信支付二维码链接
func (n *NativePaymentService) Prepay(ctx context.Context, payment domain.Payment) (string, error) {
	// 生成预支付请求
	err := n.repo.AddPayment(ctx, payment)
	if err != nil {
		return "", err
	}
	// 生成预支付链接
	resp, result, err := n.svc.Prepay(ctx, native.PrepayRequest{
		Appid:       core.String(n.appID),
		Mchid:       core.String(n.mchid),
		Description: core.String(payment.Description),
		OutTradeNo:  core.String(payment.BizTradeNo),
		NotifyUrl:   core.String(n.notifyURL),
		TimeExpire:  core.Time(time.Now().Add(30 * time.Minute)),
		Amount: &native.Amount{
			Total:    core.Int64(payment.Amt.Total),
			Currency: core.String(payment.Amt.Currency),
		},
	})
	fmt.Println("resp:", resp, "result:", result, "err:", err)
	if err != nil {
		return "", err
	}
	return *resp.CodeUrl, nil
}

// HandleCallback 处理微信支付回调
func (n *NativePaymentService) HandleCallback(ctx context.Context, txn *payments.Transaction) error {
	return n.updateByTxn(ctx, txn)
}

// updateByTxn 更新支付状态
func (n *NativePaymentService) updateByTxn(ctx context.Context, txn *payments.Transaction) error {
	// 将微信支付的状态转换为我们自己的状态
	status, ok := n.status[*txn.TradeState]

	if !ok {
		return fmt.Errorf("unknown status: %s", *txn.TradeState)
	}
	// 更新数据库支付状态，使用交易订单号、状态和微信支付交易ID更新本地数据库中的支付记录
	err := n.repo.UpdatePayment(ctx, domain.Payment{
		BizTradeNo: *txn.OutTradeNo,
		TxnID:      *txn.TransactionId,
		Status:     status,
	})

	if err != nil {
		return err
	}

	// 消息系统通知

	return nil
}

// SyncWechatInfo 同步微信支付信息
func (n *NativePaymentService) SyncWechatInfo(ctx context.Context, BizTradeNo string) error {
	// 查询微信支付订单
	// 通过业务订单号查询微信支付订单
	txn, _, err := n.svc.QueryOrderByOutTradeNo(ctx, native.QueryOrderByOutTradeNoRequest{
		OutTradeNo: core.String(BizTradeNo),
		Mchid:      core.String(n.mchid),
	})
	if err != nil {
		return err
	}
	// 更新本地支付状态
	return n.updateByTxn(ctx, txn)
}

// FindExpiredPayment 查询过期的支付记录
func (n *NativePaymentService) FindExpiredPayment(ctx context.Context, offset int, limit int, t time.Time) ([]domain.Payment, error) {
	return n.repo.FindExpiredPayments(ctx, offset, limit, t)
}

// GetPayment 根据业务交易号获取支付记录
func (n *NativePaymentService) GetPayment(ctx context.Context, bizTradeNO string) (domain.Payment, error) {
	return n.repo.GetPayment(ctx, bizTradeNO)
}