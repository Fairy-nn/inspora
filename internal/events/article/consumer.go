package article

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Fairy-nn/inspora/internal/repository"
	"github.com/IBM/sarama"
)

type Consumer interface {
	Start(ctx context.Context) error
}

// 处理从 Kafka 主题中消费到的消息
func (kc *KafkaConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	msgs := claim.Messages()
	// 遍历消息
	for msg := range msgs {
		// 处理消息
		var event ViewEvent
		err := json.Unmarshal(msg.Value, &event)
		if err != nil {
			fmt.Println("反序列化失败:", err)
			return err
		}
		// 处理事件
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		if err := kc.repo.IncrViewCount(ctx, "article", event.Aid); err != nil {
			fmt.Printf("错误: %v, 文章ID: %d, 用户ID: %d", err, event.Aid, event.Uid)
			cancel()
			return err //创建交互事件失败
		}
		cancel()                     //取消上下文
		session.MarkMessage(msg, "") //标记消息已处理
	}
	return nil //返回nil表示没有错误
}

type KafkaConsumer struct {
	client sarama.Client                             // Kafka client
	repo   repository.InteractionRepositoryInterface //交互仓库接口
}

func NewKafkaConsumer(client sarama.Client, repo repository.InteractionRepositoryInterface) Consumer {
	return &KafkaConsumer{
		client: client,
		repo:   repo,
	}
}

// Start 启动消费者组
func (kc *KafkaConsumer) Start(ctx context.Context) error {
	cg, err := sarama.NewConsumerGroupFromClient("interaction", kc.client)
	if err != nil {
		return err //创建消费者组失败
	}

	go func() {
		// 消费
		err := cg.Consume(ctx, []string{"article_view"}, kc) //消费消息
		if err != nil {
			panic(err) //消费消息失败

		}
	}()

	return nil
}

// 在消费者组开始消费之前进行一些初始化操作
func (kc *KafkaConsumer) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// 在消费者组完成消费后进行一些清理操作
func (kc *KafkaConsumer) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}
