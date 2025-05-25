package ioc

import (
	"context"
	"fmt"

	articleEvents "github.com/Fairy-nn/inspora/internal/events/article"
	events "github.com/Fairy-nn/inspora/internal/events/article"
	feedEvents "github.com/Fairy-nn/inspora/internal/events/feed"
	"github.com/IBM/sarama"
	"github.com/spf13/viper"
)

// Consumer 接口定义
type Consumer interface {
	Start(ctx context.Context) error
}

// NewConsumers 返回所有的消费者列表
func NewConsumers(articleConsumer articleEvents.Consumer, feedConsumer feedEvents.Consumer) []Consumer {
	return []Consumer{
		articleConsumer,
		feedConsumer,
	}
}

// 初始化Kafka客户端
func InitKafka() sarama.Client {
	// 读取配置文件中的Kafka地址
	type Config struct {
		Addrs []string `yaml:"addrs"`
	}
	var cfg Config
	err := viper.UnmarshalKey("kafka", &cfg)
	if err != nil {
		fmt.Println("读取配置文件失败:", err)
		panic(err) // 读取配置文件失败
	}

	// 创建Kafka客户端配置
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true // 设置生产者返回成功消息

	// 创建Kafka客户端
	client, err := sarama.NewClient(cfg.Addrs, config)
	if err != nil {
		panic(err) // 创建Kafka客户端失败
	}

	return client
}

// 初始化Kafka生产者
func NewSyncProducer(client sarama.Client) sarama.SyncProducer {
	producer, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		panic(err) // 创建Kafka生产者失败
	}
	return producer
}

// 初始化Kafka消费者
func NewSyncConsumer(client events.Consumer) []events.Consumer {
	return []events.Consumer{client}
}
