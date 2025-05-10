package payment

type PaymentEvent struct { 
	BizTradeNo string // 业务订单号
	Status     uint8 // 支付状态
}

// Topic 函数返回事件的主题
func (PaymentEvent) Topic() string {
	return "payment_event"
}
