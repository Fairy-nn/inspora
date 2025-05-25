package feed

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/repository"
	"github.com/IBM/sarama"
)

const (
	FeedTopic = "feed_events"
)

// EventTypes
const (
	EventTypeArticlePublished = "article_published" // 文章发布事件
	EventTypeArticleLiked     = "article_liked"     // 文章点赞事件
	EventTypeArticleCommented = "article_commented" // 文章评论事件
	EventTypeArticleCollected = "article_collected" // 文章收藏事件
	EventTypeUserFollowed     = "user_followed"     // 用户关注事件
)

// Producer 定义 Feed 事件生产者接口
// 负责将各种业务事件转换为 Feed 事件并发布到消息队列
type Producer interface {
	// ProduceFeedEvent 发布一个 Feed 事件
	ProduceFeedEvent(ctx context.Context, event domain.FeedEvent) error

	// 便捷方法，用于常见的事件类型
	ProduceArticlePublishedEvent(ctx context.Context, userID, articleID int64, title string) error
	ProduceArticleLikedEvent(ctx context.Context, userID, articleID, authorID int64) error
	ProduceUserFollowedEvent(ctx context.Context, followerID, followeeID int64) error
	ProduceArticleCommentedEvent(ctx context.Context, userID, articleID, authorID, commentID int64, commentContent string) error
	ProduceArticleCollectedEvent(ctx context.Context, userID, articleID, authorID int64) error
}

// KafkaProducer 基于 Kafka 实现的 Producer
type KafkaProducer struct {
	producer sarama.SyncProducer
	repo     repository.FeedRepository // 用于将事件保存到数据库
}

// NewKafkaProducer 创建一个新的 KafkaProducer
func NewKafkaProducer(producer sarama.SyncProducer, repo repository.FeedRepository) Producer {
	return &KafkaProducer{
		producer: producer,
		repo:     repo,
	}
}

// 发布一个 Feed 事件到 Kafka
func (k *KafkaProducer) ProduceFeedEvent(ctx context.Context, event domain.FeedEvent) error {
	// 记录开始生产事件的日志，包含事件类型和用户ID，便于追踪
	log.Printf("开始生产 Feed 事件: 类型=%s, 用户ID=%d", event.EventType, event.UserID)
	// 自动补全事件创建时间（若未设置）
	if event.Ctime.IsZero() {
		event.Ctime = time.Now() // 使用当前时间作为事件发生时间
	}
	// 事件持久化到数据库
	// 调用仓储层将事件保存到数据库，确保事件不丢失（实现事件溯源）
	eventID, err := k.repo.AddFeedEvent(ctx, event)
	if err != nil {
		// 记录数据库操作失败日志，包含错误详情
		log.Printf("Feed 事件保存到数据库失败: 错误=%v, 事件=%+v", err, event)
		return err // 数据库操作失败时直接返回错误，终止后续流程
	}
	// 记录数据库操作成功日志，包含事件ID
	log.Printf("Feed 事件保存到数据库成功: 事件ID=%d", eventID)
	// 将数据库生成的事件ID回填到event对象，确保消息和数据库记录关联
	event.ID = eventID

	// 事件序列化为JSON
	// 将事件对象序列化为JSON格式，以便通过Kafka传输
	value, err := json.Marshal(event)
	if err != nil {
		// 记录序列化失败日志，包含错误详情和事件内容
		log.Printf("Feed 事件序列化失败: 错误=%v, 事件=%+v", err, event)
		return err // 序列化失败时返回错误，避免发送无效消息
	}

	// 发送到Kafka消息队列
	// 构建Kafka消息，包含主题、键（用于消息分区）和值（序列化后的事件）
	partition, offset, err := k.producer.SendMessage(&sarama.ProducerMessage{
		Topic: FeedTopic,
		// 使用事件类型作为消息键，可确保相同类型事件发往同一分区（可选策略）
		Key:   sarama.StringEncoder(event.EventType),
		Value: sarama.ByteEncoder(value), // 消息体为序列化后的JSON数据
	})

	if err != nil {
		// 记录Kafka发送失败日志，包含错误详情、分区和偏移量
		log.Printf("Feed 事件发送到 Kafka 失败: 错误=%v, 主题=%s, 事件ID=%d", err, FeedTopic, event.ID)
		return err // 发送失败时返回错误，需后续重试或人工处理
	}

	// 记录成功发送到Kafka的日志，包含分区和偏移量（用于消息定位）
	log.Printf("Feed 事件成功发送到 Kafka: 类型=%s, ID=%d, 分区=%d, 偏移=%d",
		event.EventType, event.ID, partition, offset)

	return nil // 所有步骤成功完成，返回nil表示成功
}

// 发布文章发布事件
func (k *KafkaProducer) ProduceArticlePublishedEvent(ctx context.Context, userID, articleID int64, title string) error {
	log.Printf("生产文章发布事件: 用户ID=%d, 文章ID=%d, 标题=%s", userID, articleID, title)

	// 构建事件内容，包含业务相关字段（如文章ID、标题、作者ID）
	event := domain.FeedEvent{
		UserID:    userID,                    // 事件发起者为文章发布者
		EventType: EventTypeArticlePublished, // 事件类型为文章发布
		Content: map[string]interface{}{
			"article_id": articleID, // 文章ID
			"title":      title,     // 文章标题
			"author_id":  userID,    // 作者ID（与userID一致）
		},
		Ctime: time.Now(), // 使用当前时间作为事件创建时间
	}

	// 调用核心发布方法，将事件发送到Kafka并保存到数据库
	return k.ProduceFeedEvent(ctx, event)
}

// 发布文章点赞事件
func (k *KafkaProducer) ProduceArticleLikedEvent(ctx context.Context, userID, articleID, authorID int64) error {
	log.Printf("生产文章点赞事件: 点赞用户ID=%d, 文章ID=%d, 作者ID=%d", userID, articleID, authorID)

	event := domain.FeedEvent{
		UserID:    userID,                // 事件发起者为点赞用户
		EventType: EventTypeArticleLiked, // 事件类型为点赞
		Content: map[string]interface{}{
			"article_id": articleID, // 被点赞的文章ID
			"liker_id":   userID,    // 点赞者ID
			"author_id":  authorID,  // 文章作者ID（用于后续通知）
		},
		Ctime: time.Now(),
	}

	return k.ProduceFeedEvent(ctx, event)
}

func (k *KafkaProducer) ProduceUserFollowedEvent(ctx context.Context, followerID, followeeID int64) error {
	log.Printf("生产用户关注事件: 关注者ID=%d, 被关注者ID=%d", followerID, followeeID)

	event := domain.FeedEvent{
		UserID:    followerID,            // 事件发起者为关注者
		EventType: EventTypeUserFollowed, // 事件类型为关注
		Content: map[string]interface{}{
			"follower_id": followerID, // 关注者ID
			"followee_id": followeeID, // 被关注者ID
		},
		Ctime: time.Now(),
	}

	return k.ProduceFeedEvent(ctx, event)
}

func (k *KafkaProducer) ProduceArticleCommentedEvent(ctx context.Context, userID, articleID, authorID, commentID int64, commentContent string) error {
	log.Printf("生产文章评论事件: 评论用户ID=%d, 文章ID=%d, 作者ID=%d, 评论ID=%d", userID, articleID, authorID, commentID)

	event := domain.FeedEvent{
		UserID:    userID,                    // 事件发起者为评论用户
		EventType: EventTypeArticleCommented, // 事件类型为评论
		Content: map[string]interface{}{
			"article_id":      articleID,      // 被评论的文章ID
			"comment_id":      commentID,      // 评论ID
			"comment_content": commentContent, // 评论内容
			"author_id":       authorID,       // 文章作者ID（用于后续通知）
		},
		Ctime: time.Now(),
	}

	return k.ProduceFeedEvent(ctx, event)
}

func (k *KafkaProducer) ProduceArticleCollectedEvent(ctx context.Context, userID, articleID, authorID int64) error {
	log.Printf("生产文章收藏事件: 收藏用户ID=%d, 文章ID=%d, 作者ID=%d", userID, articleID, authorID)

	event := domain.FeedEvent{
		UserID:    userID,                    // 事件发起者为收藏用户
		EventType: EventTypeArticleCollected, // 事件类型为收藏
		Content: map[string]interface{}{
			"article_id":   articleID, // 被收藏的文章ID
			"collector_id": userID,    // 收藏者ID
			"author_id":    authorID,  // 文章作者ID（用于后续通知）
		},
		Ctime: time.Now(),
	}

	return k.ProduceFeedEvent(ctx, event)
}
