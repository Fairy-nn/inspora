package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Fairy-nn/inspora/internal/service"
	"github.com/IBM/sarama"
)

type PaymentEventConsumer struct {
	client sarama.Client                  // Sarama客户端，用于与Kafka交互
	svc    service.RewardServiceInterface // 奖励服务接口，用于更新奖励状态
}

func (r *PaymentEventConsumer) Start() error {
	cg, err := sarama.NewConsumerGroupFromClient("reward", r.client)
	if err != nil {
		return err
	}
	go func() {
		err := cg.Consume(context.Background(), []string{"payment_events"}, r)
		if err != nil {
			fmt.Println("Error consuming messages:", err)
		}
	}()
	return err
}

func (r *PaymentEventConsumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}
func (r *PaymentEventConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// 处理消费到的消息
func (r *PaymentEventConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var evt PaymentEvent
		err := json.Unmarshal(msg.Value, &evt)
		if err != nil {
			fmt.Println("Failed to unmarshal message:", err)
			session.MarkMessage(msg, "")
			continue
		}

		err = r.Consume(msg, evt)
		if err != nil {
			fmt.Println("Failed to consume message:", err)
		}

		session.MarkMessage(msg, "")
	}
	return nil
}

// Consume 处理单个支付事件消息
func (r *PaymentEventConsumer) Consume(
	msg *sarama.ConsumerMessage,
	evt PaymentEvent) error {
	if !strings.HasPrefix(evt.BizTradeNo, "reward") {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	return r.svc.UpdateReward(ctx, evt.BizTradeNo, evt.ToRewardDomainStatus())
}
