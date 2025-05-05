package article

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IBM/sarama"
)

type Producer interface {
	ProducerViewEvent(ctx context.Context, event ViewEvent) error
}

type KafkaProducer struct {
	producer sarama.SyncProducer
}

type ViewEvent struct {
	Uid int64 // 用户ID
	Aid int64 // 文章ID
}

func NewKafkaProducer(pc sarama.SyncProducer) Producer {
	return &KafkaProducer{
		producer: pc,
	}
}

func (kp *KafkaProducer) ProducerViewEvent(ctx context.Context, event ViewEvent) error {
	// 将事件序列化为JSON格式
	data, err := json.Marshal(event)
	// fmt.Printf("生产者序列化: %s\n", string(data))
	fmt.Printf("生产者事件: %v\n", event)
	if err != nil {
		return err
	}
	// 发送消息到Kafka主题，主题名称为"article_view"
	_, _, err = kp.producer.SendMessage(&sarama.ProducerMessage{
		Topic: "article_view",
		Value: sarama.ByteEncoder(data),
	})
	return err
}
