package payment

import (
	"context"
	"encoding/json"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/IBM/sarama"
)

type PaymentProducerInterface interface {
	// ProducePaymentEvent 生产支付事件
	ProducePaymentEvent(ctx context.Context, evt PaymentEvent) error
}

type SaramaPaymentProducer struct {
	producer sarama.SyncProducer
}

func NewSaramaPaymentProducer(producer sarama.SyncProducer) PaymentProducerInterface {
	return &SaramaPaymentProducer{producer: producer}
}

// ProducePaymentEvent 生产支付事件
func (s *SaramaPaymentProducer) ProducePaymentEvent(ctx context.Context, evt PaymentEvent) error {
	// 数据序列化
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	// 发送消息，这里使用了同步发送
	_, _, err = s.producer.SendMessage(&sarama.ProducerMessage{
		Key:   sarama.StringEncoder(evt.BizTradeNo),
		Topic: evt.Topic(),
		Value: sarama.ByteEncoder(data),
	})
	return err
}
// 将 payment 的状态转化为 reward 的状态
func (p PaymentEvent) ToRewardDomainStatus() domain.RewardStatus {
	switch p.Status {
	case 1:
		return domain.RewardStatusInit
	case 2:
		return domain.RewardStatusPaid
	case 3, 4:
		return domain.RewardStatusFailed
	}
	return domain.RewardStatusUnknown
}
