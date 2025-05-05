package article

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Fairy-nn/inspora/internal/repository"
	"github.com/IBM/sarama"
)

type InteractionBatchConsumer struct {
	// Kafka 客户端，用于连接和操作 Kafka 集群
	client sarama.Client
	// 交互信息仓库接口，用于处理与交互信息相关的数据库操作
	repo repository.InteractionRepositoryInterface
}

func NewInteractionBatchConsumer(client sarama.Client, repo repository.InteractionRepositoryInterface) Consumer {
	return &InteractionBatchConsumer{
		client: client,
		repo:   repo,
	}
}

func (b *InteractionBatchConsumer) Start(ctx context.Context) error {
	// 创建消费者组
	cg, err := sarama.NewConsumerGroupFromClient("interaction", b.client)
	if err != nil {
		return err // 创建消费者组失败
	}

	// 启动一个新的 goroutine来处理消息消费
	go func() {
		err := cg.Consume(ctx, []string{"article_view"}, b) // 消费消息
		if err != nil {
			panic(err) // 消费消息失败
		}
	}()
	return nil
}

func (k *InteractionBatchConsumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (k *InteractionBatchConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (k *InteractionBatchConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	msgsCh := claim.Messages()
	batch_size := 100 // 批量大小

	// 持续处理消息
	for {
		// 用于控制消息接收的时间
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		done := false
		msgs := make([]*sarama.ConsumerMessage, 0, batch_size) // 消息切片
		ts := make([]ViewEvent, 0, batch_size)                 // 事件切片

		// 循环接收消息,直到达到批量大小或超时
		for i := 0; i < batch_size && !done; i++ {
			select {
			case <-ctx.Done():
				done = true // 超时，退出循环
			case msg, ok := <-msgsCh:
				if !ok {
					cancel() // 关闭消息通道，退出循环
					return sarama.ErrCannotTransitionNilError
				}
				// 解析消息
				var event ViewEvent
				err := json.Unmarshal(msg.Value, &event)
				if err != nil {
					fmt.Println("解析消息失败:", err)
					continue // 解析失败，继续下一个消息
				}
				ts = append(ts, event)   // 将事件添加到切片中
				msgs = append(msgs, msg) // 将消息添加到切片中
			}
		}
		cancel() // 取消上下文
		if len(msgs) == 0 {
			continue // 没有消息，继续下一个批次
		}

		ids := make([]int64, 0, len(ts))   // ID切片
		bizs := make([]string, 0, len(ts)) // 业务切片
		for _, v := range ts {
			ids = append(ids, v.Aid)       // 添加文章ID
			bizs = append(bizs, "article") // 添加业务类型
		}
		// 批量处理消息
		ctx_1, cancel_1 := context.WithTimeout(context.Background(), time.Second*5)

		err := k.repo.BatchIncrViewCount(ctx_1, bizs, ids) // 批量增加文章的浏览量

		if err != nil {
			fmt.Println("批量处理消息失败:", err)
			cancel_1() // 取消上下文
			continue
		}
		cancel_1()

		// 标记消息已处理
		for _, m := range msgs {
			session.MarkMessage(m, "")
		}
	}
}
