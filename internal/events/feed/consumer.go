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
	// ConsumerGroupID 是 Feed 消费者组的 ID
	ConsumerGroupID = "feed_consumer_group"
	// MaxFanoutBatchSize 是每批处理的最大粉丝数量
	MaxFanoutBatchSize = 1000
	// MaxFollowerForPush 是使用推模型的最大粉丝数量阈值
	// 如果粉丝数量超过此值，将不会直接推送到所有粉丝的收件箱
	MaxFollowerForPush = 100000
)

// FeedConsumer 是 Feed 事件消费者接口
type Consumer interface {
	// Start 启动消费者
	Start(ctx context.Context) error
}

// KafkaFeedConsumer 是基于 Kafka 的 Feed 事件消费者
type KafkaFeedConsumer struct {
	client      sarama.ConsumerGroup
	feedRepo    repository.FeedRepository
	followRepo  repository.FollowRepository
	articleRepo repository.ArticleRepository
	userRepo    repository.UserRepositoryInterface
}

// NewKafkaFeedConsumer 创建一个新的 KafkaFeedConsumer
func NewKafkaFeedConsumer(
	client sarama.Client,
	feedRepo repository.FeedRepository,
	followRepo repository.FollowRepository,
	articleRepo repository.ArticleRepository,
	userRepo repository.UserRepositoryInterface,
) Consumer {
	consumerGroup, err := sarama.NewConsumerGroupFromClient(ConsumerGroupID, client)
	if err != nil {
		panic(err)
	}

	return &KafkaFeedConsumer{
		client:      consumerGroup,
		feedRepo:    feedRepo,
		followRepo:  followRepo,
		articleRepo: articleRepo,
		userRepo:    userRepo,
	}
}

// Start 启动消费者
func (k *KafkaFeedConsumer) Start(ctx context.Context) error {
	log.Printf("Feed 消费者开始启动，订阅主题: %s", FeedTopic)
	topics := []string{FeedTopic}
	for {
		select {
		case <-ctx.Done():
			log.Printf("Feed 消费者收到停止信号，即将退出")
			return ctx.Err()
		default:
			log.Printf("Feed 消费者开始一轮消费")
			err := k.client.Consume(ctx, topics, k)
			if err != nil {
				log.Printf("Feed 消费错误: %v，将在5秒后重试", err)
				time.Sleep(time.Second * 5) // 出错后等待一段时间再重试
			}
		}
	}
}

// Setup 是在消费者会话开始前调用的钩子
func (k *KafkaFeedConsumer) Setup(sarama.ConsumerGroupSession) error {
	log.Printf("Feed 消费者会话已初始化")
	return nil
}

// Cleanup 是在消费者会话结束后调用的钩子
func (k *KafkaFeedConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	log.Printf("Feed 消费者会话已清理")
	return nil
}

// ConsumeClaim 处理从 Kafka 接收到的消息
func (k *KafkaFeedConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	log.Printf("Feed 消费者开始处理分区 %d 的消息", claim.Partition())
	for msg := range claim.Messages() {
		log.Printf("收到新的 Feed 事件消息: 偏移=%d, 分区=%d", msg.Offset, msg.Partition)

		if err := k.processFeedEvent(session.Context(), msg); err != nil {
			log.Printf("处理 Feed 事件失败: %v", err)
		} else {
			log.Printf("成功处理 Feed 事件消息: 偏移=%d, 分区=%d", msg.Offset, msg.Partition)
		}

		session.MarkMessage(msg, "")
	}
	return nil
}

// processFeedEvent 处理单个 Feed 事件
func (k *KafkaFeedConsumer) processFeedEvent(ctx context.Context, msg *sarama.ConsumerMessage) error {
	// 解析事件
	var event domain.FeedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		log.Printf("解析 Feed 事件失败: %v", err)
		return err
	}

	log.Printf("🔍 处理 Feed 事件: 类型=%s, ID=%d, 用户ID=%d", event.EventType, event.ID, event.UserID)

	switch event.EventType {
	case EventTypeArticlePublished:
		log.Printf("开始处理文章发布事件: ID=%d", event.ID)
		err := k.handleArticlePublished(ctx, event)
		if err != nil {
			log.Printf(" 文章发布事件处理失败: %v", err)
		} else {
			log.Printf("文章发布事件处理成功: ID=%d", event.ID)
		}
		return err

	case EventTypeArticleLiked:
		log.Printf("开始处理文章点赞事件: ID=%d", event.ID)
		err := k.handleArticleLiked(ctx, event)
		if err != nil {
			log.Printf("文章点赞事件处理失败: %v", err)
		} else {
			log.Printf("文章点赞事件处理成功: ID=%d", event.ID)
		}
		return err

	case EventTypeUserFollowed:
		log.Printf("开始处理用户关注事件: ID=%d", event.ID)
		err := k.handleUserFollowed(ctx, event)
		if err != nil {
			log.Printf("用户关注事件处理失败: %v", err)
		} else {
			log.Printf("用户关注事件处理成功: ID=%d", event.ID)
		}
		return err

	case EventTypeArticleCommented:
		log.Printf(" 开始处理文章评论事件: ID=%d", event.ID)
		err := k.handleArticleCommented(ctx, event)
		if err != nil {
			log.Printf("文章评论事件处理失败: %v", err)
		} else {
			log.Printf("文章评论事件处理成功: ID=%d", event.ID)
		}
		return err

	case EventTypeArticleCollected:
		log.Printf("开始处理文章收藏事件: ID=%d", event.ID)
		err := k.handleArticleCollected(ctx, event)
		if err != nil {
			log.Printf(" 文章收藏事件处理失败: %v", err)
		} else {
			log.Printf("文章收藏事件处理成功: ID=%d", event.ID)
		}
		return err

	default:
		// 未知事件类型，忽略
		log.Printf("收到未知事件类型: %s, 已忽略", event.EventType)
		return nil
	}
}

// handleArticlePublished 处理文章发布事件
func (k *KafkaFeedConsumer) handleArticlePublished(ctx context.Context, event domain.FeedEvent) error {
	// 从事件中获取信息
	authorID := event.UserID
	articleID, ok := event.Content["article_id"].(float64)
	if !ok {
		log.Printf("文章发布事件格式不正确: article_id 不存在或格式错误")
		return nil // 跳过格式不正确的事件
	}

	log.Printf("处理文章发布事件: 作者ID=%d, 文章ID=%d", authorID, int64(articleID))

	// 创建 FeedItem
	item, err := domain.NewUserFeedItem(
		"article",
		int64(articleID),
		authorID,
		event.Ctime,
		map[string]interface{}{
			"type":       "article_published",
			"article_id": articleID,
			"author_id":  authorID,
		},
	)
	if err != nil {
		log.Printf("创建 Feed 项失败: %v", err)
		return err
	}

	if err := k.feedRepo.AddToOutbox(ctx, authorID, item); err != nil {
		log.Printf("添加到发件箱失败: %v", err)
		return err
	}
	log.Printf("已添加到作者(ID=%d)的发件箱", authorID)

	// 2. 获取作者的粉丝数量
	followerStats, err := k.followRepo.GetStatistics(ctx, authorID)
	if err != nil {
		log.Printf("获取粉丝统计失败: %v", err)
		return err
	}
	log.Printf("作者(ID=%d)的粉丝数量: %d", authorID, followerStats.Followers)

	// 如果粉丝数量超过阈值，则不使用推模型，依靠拉模型
	if followerStats.Followers > MaxFollowerForPush {
		// 为大V用户只推送给一部分活跃粉丝，或者完全不推送
		log.Printf("用户 %d 是大V (粉丝: %d)，不进行全面推送", authorID, followerStats.Followers)
		return nil
	}

	// 3. 获取作者的粉丝列表并推送
	offset := int64(0)
	limit := int64(MaxFanoutBatchSize)

	totalFanoutCount := 0
	for {
		followers, err := k.followRepo.GetFollowerList(ctx, authorID, offset, limit)
		if err != nil {
			log.Printf("获取粉丝列表失败: %v", err)
			return err
		}

		if len(followers) == 0 {
			log.Printf("没有更多粉丝，推送结束")
			break // 没有更多粉丝了
		}

		log.Printf("开始向 %d 名粉丝推送 Feed", len(followers))

		// 批量推送到粉丝的收件箱
		successCount := 0
		for _, follower := range followers {
			if err := k.feedRepo.AddToInbox(ctx, follower.Follower, item); err != nil {
				log.Printf("推送到用户 %d 的收件箱失败: %v", follower.Follower, err)
				continue
			}
			successCount++
		}

		totalFanoutCount += successCount
		log.Printf("成功推送给 %d/%d 名粉丝", successCount, len(followers))

		if len(followers) < int(limit) {
			log.Printf("粉丝批次不足，推送结束")
			break // 没有更多粉丝了
		}

		offset += limit
	}

	log.Printf("文章发布事件处理完成，共推送给 %d 名粉丝", totalFanoutCount)

	return nil
}

// handleArticleLiked 处理文章点赞事件
func (k *KafkaFeedConsumer) handleArticleLiked(ctx context.Context, event domain.FeedEvent) error {
	// 从事件中获取信息
	likerID := event.UserID
	articleID, _ := event.Content["article_id"].(float64)
	authorID, _ := event.Content["author_id"].(float64)

	// 创建 FeedItem
	item, err := domain.NewUserFeedItem(
		"like",
		int64(articleID),
		likerID,
		event.Ctime,
		map[string]interface{}{
			"type":       "article_liked",
			"article_id": articleID,
			"liker_id":   likerID,
			"author_id":  authorID,
		},
	)
	if err != nil {
		return err
	}

	// 将事件添加到被点赞文章作者的收件箱 (仅通知作者)
	return k.feedRepo.AddToInbox(ctx, int64(authorID), item)
}

// handleUserFollowed 处理用户关注事件
func (k *KafkaFeedConsumer) handleUserFollowed(ctx context.Context, event domain.FeedEvent) error {
	// 从事件中获取信息
	followerID := event.UserID
	followeeID, _ := event.Content["followee_id"].(float64)

	// 创建 FeedItem
	item, err := domain.NewUserFeedItem(
		"follow",
		int64(followeeID), // 使用被关注者ID作为项目ID
		followerID,
		event.Ctime,
		map[string]interface{}{
			"type":        "user_followed",
			"follower_id": followerID,
			"followee_id": followeeID,
		},
	)
	if err != nil {
		return err
	}

	if err := k.feedRepo.AddToInbox(ctx, int64(followeeID), item); err != nil {
		return err
	}

	return nil
}

// handleArticleCommented 处理文章评论事件
func (k *KafkaFeedConsumer) handleArticleCommented(ctx context.Context, event domain.FeedEvent) error {
	// 从事件中获取信息
	commenterID := event.UserID
	articleID, _ := event.Content["article_id"].(float64)
	commentID, _ := event.Content["comment_id"].(float64)
	commentContent, _ := event.Content["comment_content"].(string)
	authorID, _ := event.Content["author_id"].(float64)

	// 截取评论内容 (如果太长)
	if len(commentContent) > 50 {
		commentContent = commentContent[:50] + "..."
	}

	// 创建 FeedItem
	item, err := domain.NewUserFeedItem(
		"comment",
		int64(commentID),
		commenterID,
		event.Ctime,
		map[string]interface{}{
			"type":            "article_commented",
			"article_id":      articleID,
			"commenter_id":    commenterID,
			"author_id":       authorID,
			"comment_id":      commentID,
			"comment_content": commentContent,
		},
	)
	if err != nil {
		return err
	}

	// 将评论事件添加到被评论文章作者的收件箱 (仅通知作者)
	// 如果评论者就是作者自己，则不需要额外通知
	if commenterID != int64(authorID) {
		return k.feedRepo.AddToInbox(ctx, int64(authorID), item)
	}

	return nil
}

// handleArticleCollected 处理文章收藏事件
func (k *KafkaFeedConsumer) handleArticleCollected(ctx context.Context, event domain.FeedEvent) error {
	// 从事件中获取信息
	collectorID := event.UserID
	articleID, _ := event.Content["article_id"].(float64)
	authorID, _ := event.Content["author_id"].(float64)

	// 创建 FeedItem
	item, err := domain.NewUserFeedItem(
		"collect",
		int64(articleID),
		collectorID,
		event.Ctime,
		map[string]interface{}{
			"type":         "article_collected",
			"article_id":   articleID,
			"collector_id": collectorID,
			"author_id":    authorID,
		},
	)
	if err != nil {
		return err
	}

	// 将收藏事件添加到文章作者的收件箱 (仅通知作者)
	// 如果收藏者就是作者自己，则不需要额外通知
	if collectorID != int64(authorID) {
		return k.feedRepo.AddToInbox(ctx, int64(authorID), item)
	}

	return nil
}
