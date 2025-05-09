package domain

import "github.com/wechatpay-apiv3/wechatpay-go/services/payments"

type Amount struct {
	Currency string // 货币类型
	Total    int64  // 支付金额
}

type Payment struct {
	Amt         Amount        // 支付金额
	BizTradeNo  string        // 业务交易号
	Description string        // 支付描述
	Status      PaymentStatus // 支付状态
	TxnID       string        // 交易ID
}

type PaymentStatus uint8

// 将 PaymentStatus 转换为 uint8 类型
func (s PaymentStatus) AsUint8() uint8 {
	return uint8(s)
}

const (
	PaymentStatusUnknown = iota
	PaymentStatusInit
	PaymentStatusSuccess
	PaymentStatusFailed
	PaymentStatusRefund
)

type Txn = payments.Transaction
