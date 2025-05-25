package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/events/feed"
	"github.com/Fairy-nn/inspora/internal/repository"
)

// FeedServiceInterface 定义了 Feed 服务的接口
type FeedServiceInterface interface {
	// GetUserFeed 获取用户的 Feed 流
	GetUserFeed(ctx context.Context, userID int64, offset, limit int) ([]domain.RenderedFeedItem, error)
	// PublishArticle 发布文章到 Feed
	PublishArticle(ctx context.Context, userID, articleID int64, title string) error
	// LikeArticle 点赞文章，产生 Feed 事件
	LikeArticle(ctx context.Context, userID, articleID, authorID int64) error
	// FollowUser 关注用户，产生 Feed 事件
	FollowUser(ctx context.Context, followerID, followeeID int64) error
	// CommentArticle 评论文章，产生 Feed 事件
	CommentArticle(ctx context.Context, userID, articleID, authorID, commentID int64, commentContent string) error
	// CollectArticle 收藏文章，产生 Feed 事件
	CollectArticle(ctx context.Context, userID, articleID, authorID int64) error
	// RebuildUserFeed 重建用户的Feed (包括收件箱和发件箱)
	RebuildUserFeed(ctx context.Context, userID int64, sinceDays int) error
}

// FeedService 实现 FeedServiceInterface 接口，负责处理用户 Feed 流相关业务逻辑
// 包含 Feed 推送、获取、排序等核心功能
type FeedService struct {
	feedRepo   repository.FeedRepository
	followRepo repository.FollowRepository        // 管理用户关注关系的仓储层接口
	articleSvc ArticleServiceInterface            // 文章服务接口，用于获取文章详情
	userRepo   repository.UserRepositoryInterface // 用户信息仓储层接口
	// Kafka 生产者，用于异步推送 Feed 事件
	feedProd feed.Producer
	// 判定大V用户的粉丝数阈值（超过此值视为大V）
	bigVThreshold int64
	// 用户收件箱最大长度，超出部分将被淘汰
	inboxMaxLen int
}

func NewFeedService(
	feedRepo repository.FeedRepository,
	followRepo repository.FollowRepository,
	articleSvc ArticleServiceInterface,
	userRepo repository.UserRepositoryInterface,
	feedProd feed.Producer) FeedServiceInterface {
	// 初始化并返回服务实例
	return &FeedService{
		feedRepo:      feedRepo,             // 注入 Feed 仓储
		followRepo:    followRepo,           // 注入关注关系仓储
		articleSvc:    articleSvc,           // 注入文章服务
		userRepo:      userRepo,             // 注入用户信息仓储
		feedProd:      feedProd,             // 注入 Kafka 生产者
		bigVThreshold: 10000, // 配置大V阈值
		inboxMaxLen:   1000,   // 配置收件箱最大长度
	}
}

// GetUserFeed 获取用户的 Feed 流
// 采用推拉混合模式：
//   - 普通用户内容：通过推模式预先写入收件箱
//   - 大V用户内容：通过拉模式实时获取（避免大V内容推送给过多粉丝导致性能问题）
func (s *FeedService) GetUserFeed(ctx context.Context, userID int64, offset, limit int) ([]domain.RenderedFeedItem, error) {
	// 从收件箱中获取基础 Feed（普通用户推送给当前用户的内容）
	inboxItems, err := s.feedRepo.GetInboxForUser(ctx, userID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("获取收件箱失败: %w", err)
	}
	// 获取用户的关注列表
	// 假设用户最多关注100个用户
	followeeList, err := s.followRepo.GetFolloweeList(ctx, userID, 0, 100)
	if err != nil {
		return nil, fmt.Errorf("获取关注列表失败: %w", err)
	}

	// 找出大V用户（粉丝数超过阈值的用户）
	var bigVIDs []int64
	for _, followee := range followeeList {
		// 获取关注对象的粉丝统计
		stats, err := s.followRepo.GetStatistics(ctx, followee.Followee)
		if err != nil {
			// 忽略获取失败的用户，继续处理其他用户
			continue
		}

		// 判断是否为大V（粉丝数超过阈值）
		if stats.Followers > s.bigVThreshold {
			bigVIDs = append(bigVIDs, followee.Followee)
		}
	}

	// 从大V的发件箱中拉取内容（拉模式核心逻辑）
	// 大V内容不预先推送，而是在用户请求时实时拉取
	var bigVItems []domain.UserFeedItem
	for _, bigVID := range bigVIDs {
		// 从每个大V获取最近的10条Feed（可根据业务调整数量）
		items, err := s.feedRepo.GetOutboxForUser(ctx, bigVID, 0, 10)
		if err != nil {
			// 忽略获取失败的大V，继续处理其他大V
			continue
		}
		bigVItems = append(bigVItems, items...)
	}

	// 合并收件箱（普通用户推送）和大V发件箱（实时拉取）的内容
	allItems := append(inboxItems, bigVItems...)

	// 将 Feed 项目转换为可展示的形式（添加用户信息、格式化内容等）
	renderedItems := make([]domain.RenderedFeedItem, 0, len(allItems))
	for _, item := range allItems {
		rendered, err := s.renderFeedItem(ctx, item)
		if err != nil {
			// 渲染失败则跳过该项目（保证主流程不受影响）
			// 注：生产环境建议记录错误日志以便排查
			continue
		}
		renderedItems = append(renderedItems, rendered)
	}

	// 异步整理用户的收件箱（防止收件箱无限增长）
	// 使用单独 goroutine 执行，避免影响主响应流程
	go func() {
		// 设置超时上下文，防止长时间阻塞
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 修剪收件箱，保持最大长度限制
		if err := s.feedRepo.TrimInbox(ctx, userID, s.inboxMaxLen); err != nil {
			// 记录错误但不影响主流程（异步任务失败不影响用户体验）
			fmt.Printf("整理收件箱失败: %v\n", err)
		}
	}()

	return renderedItems, nil
}

// renderFeedItem 渲染 Feed 项目为可展示的形式
func (s *FeedService) renderFeedItem(ctx context.Context, item domain.UserFeedItem) (domain.RenderedFeedItem, error) {
	// 1. 解析 Feed 内容（JSON 字符串反序列化为 map）
	var content map[string]interface{}
	if err := json.Unmarshal([]byte(item.Content), &content); err != nil {
		return domain.RenderedFeedItem{}, fmt.Errorf("解析 Feed 内容失败: %w", err)
	}
	// 2. 获取用户信息（发件人）
	user, err := s.userRepo.GetByID(ctx, item.ActorID)
	if err != nil {
		return domain.RenderedFeedItem{}, fmt.Errorf("获取用户信息失败: %w", err)
	}
	// 3.初始化渲染结果，设置公共字段
	rendered := domain.RenderedFeedItem{
		ID:        item.ItemID,    // Feed 项唯一标识
		ActorName: user.Email,     // 显示名称（使用邮箱，实际应改为用户名）
		ActorID:   user.ID,        // 动作发起者 ID
		Timestamp: item.Timestamp, // 事件时间戳（用于排序）
	}
	// 4. 根据内容类型设置具体渲染字段
	switch item.ItemType {
	case "article":
		// 文章发布事件
		articleID, ok := content["article_id"].(int64)
		if !ok {
			return domain.RenderedFeedItem{}, fmt.Errorf("文章 ID 类型错误")
		}
		article, err := s.articleSvc.FindById(ctx, articleID, item.ActorID)
		if err != nil {
			return domain.RenderedFeedItem{}, fmt.Errorf("获取文章信息失败: %w", err)
		}
		// 设置文章类型feed的渲染字段
		rendered.Verb = "published"                            // 动作描述（发布）
		rendered.Object = "article"                            // 对象类型
		rendered.ObjectID = article.ID                         // 对象 ID
		rendered.Title = article.Title                         // 文章标题
		rendered.Summary = truncateText(article.Content, 100)  // 截取内容前100个字符作为摘要
		rendered.Link = fmt.Sprintf("/article/%d", article.ID) // 文章详情链接
	case "like":
		// 点赞事件
		articleID, ok := content["article_id"].(int64)
		if !ok {
			return domain.RenderedFeedItem{}, fmt.Errorf("文章 ID 类型错误")
		}
		// 获取文章信息
		article, err := s.articleSvc.FindById(ctx, articleID, item.ActorID)
		if err != nil {
			return domain.RenderedFeedItem{}, fmt.Errorf("获取文章信息失败: %w", err)
		}
		// 设置点赞类型feed的渲染字段
		rendered.Verb = "liked"                                                    // 动作描述（点赞）
		rendered.Object = "article"                                                // 对象类型
		rendered.ObjectID = article.ID                                             // 对象 ID
		rendered.Title = article.Title                                             // 文章标题
		rendered.Summary = fmt.Sprintf("%s 点赞了文章 《%s》", user.Email, article.Title) // 摘要文案
		rendered.Link = fmt.Sprintf("/article/%d", article.ID)                     // 文章详情链接
	case "follow":
		// 关注事件
		followeeID, ok := content["followee_id"].(int64)
		if !ok {
			return domain.RenderedFeedItem{}, fmt.Errorf("关注对象 ID 类型错误")
		}
		// 获取关注对象信息
		followee, err := s.userRepo.GetByID(ctx, followeeID)
		if err != nil {
			return domain.RenderedFeedItem{}, fmt.Errorf("获取关注对象信息失败: %w", err)
		}
		// 设置关注类型feed的渲染字段
		rendered.Verb = "followed"                                              // 动作描述（关注）
		rendered.Object = "user"                                                // 对象类型
		rendered.ObjectID = followee.ID                                         // 对象 ID
		rendered.Title = followee.Email                                         // 关注对象名称
		rendered.Summary = fmt.Sprintf("%s 关注了 %s", user.Email, followee.Email) // 摘要文案
		rendered.Link = fmt.Sprintf("/user/%d", followee.ID)                    // 关注对象链接
	case "comment":
		// 评论事件
		articleID, ok := content["article_id"].(int64)
		if !ok {
			return domain.RenderedFeedItem{}, fmt.Errorf("文章 ID 类型错误")
		}
		commentContent, _ := content["comment"].(string)
		// 获取文章信息
		article, err := s.articleSvc.FindById(ctx, articleID, item.ActorID)
		if err != nil {
			return domain.RenderedFeedItem{}, fmt.Errorf("获取文章信息失败: %w", err)
		}
		// 设置评论类型feed的渲染字段
		rendered.Verb = "commented"                                                                    // 动作描述（评论）
		rendered.Object = "article"                                                                    // 对象类型
		rendered.ObjectID = article.ID                                                                 // 对象 ID
		rendered.Title = article.Title                                                                 // 文章标题
		rendered.Summary = fmt.Sprintf("%s 评论了文章 《%s》: %s", user.Email, article.Title, commentContent) // 摘要文案
		rendered.Link = fmt.Sprintf("/article/%d", article.ID)                                         // 文章详情链接
	case "collect":
		// 收藏事件
		articleID, ok := content["article_id"].(int64)
		if !ok {
			return domain.RenderedFeedItem{}, fmt.Errorf("文章 ID 类型错误")
		}
		// 获取文章信息
		article, err := s.articleSvc.FindById(ctx, articleID, item.ActorID)
		if err != nil {
			return domain.RenderedFeedItem{}, fmt.Errorf("获取文章信息失败: %w", err)
		}
		// 设置收藏类型feed的渲染字段
		rendered.Verb = "collected"                                                // 动作描述（收藏）
		rendered.Object = "article"                                                // 对象类型
		rendered.ObjectID = article.ID                                             // 对象 ID
		rendered.Title = article.Title                                             // 文章标题
		rendered.Summary = fmt.Sprintf("%s 收藏了文章 《%s》", user.Email, article.Title) // 摘要文案
		rendered.Link = fmt.Sprintf("/article/%d", article.ID)                     // 文章详情链接
	default:
		return domain.RenderedFeedItem{}, fmt.Errorf("未知的 Feed 项类型: %s", item.ItemType)
	}

	return rendered, nil
}

// truncateText 截取字符串到指定长度，并添加省略号
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text // 长度未超过限制，直接返回原字符串
	}
	return text[:maxLen] + "..." // 截断并添加省略号
}

// RebuildUserFeed 重建用户的Feed (包括收件箱和发件箱)
func (s *FeedService) RebuildUserFeed(ctx context.Context, userID int64, sinceDays int) error {
	// 1. 清空用户的收件箱和发件箱
	inboxKey := fmt.Sprintf("feed:inbox:%d", userID)
	outboxKey := fmt.Sprintf("feed:outbox:%d", userID)

	client := s.feedRepo.(*repository.CachedFeedRepository).GetRedisClient()
	if client == nil {
		return errors.New("无法获取Redis客户端")
	}

	// 删除收件箱和发件箱
	if err := client.Del(ctx, inboxKey, outboxKey).Err(); err != nil {
		return fmt.Errorf("清空收件箱和发件箱失败: %w", err)
	}

	// 2. 计算起始时间
	since := time.Now().AddDate(0, 0, -sinceDays)

	// 3. 重建发件箱 - 获取用户自己产生的事件
	userEvents, err := s.rebuildUserOutbox(ctx, userID, since)
	if err != nil {
		return fmt.Errorf("重建发件箱失败: %w", err)
	}

	// 4. 重建收件箱 - 获取用户关注的人的事件
	if err := s.rebuildUserInbox(ctx, userID, since, userEvents); err != nil {
		return fmt.Errorf("重建收件箱失败: %w", err)
	}

	return nil
}

// rebuildUserOutbox 重建用户的发件箱
func (s *FeedService) rebuildUserOutbox(ctx context.Context, userID int64, since time.Time) (map[int64]bool, error) {
	offset := 0
	limit := 100
	processedEvents := make(map[int64]bool)

	for {
		// 获取用户的Feed事件
		events, err := s.feedRepo.GetFeedEventsSince(ctx, since, offset, limit)
		if err != nil {
			return nil, err
		}

		// 如果没有更多事件，退出循环
		if len(events) == 0 {
			break
		}

		// 处理事件
		for _, event := range events {
			// 只处理该用户发起的事件
			if event.UserID != userID {
				continue
			}

			// 记录已处理的事件ID
			processedEvents[event.ID] = true

			// 根据事件类型生成FeedItem并添加到发件箱
			feedItem, err := s.createFeedItemFromEvent(event)
			if err != nil {
				// 记录错误但继续处理
				fmt.Printf("从事件创建Feed项失败 (ID:%d): %v\n", event.ID, err)
				continue
			}

			// 添加到发件箱
			if err := s.feedRepo.AddToOutbox(ctx, userID, feedItem); err != nil {
				// 记录错误但继续处理
				fmt.Printf("添加到发件箱失败 (ID:%d): %v\n", event.ID, err)
			}
		}

		// 更新偏移量
		offset += len(events)

		// 如果获取的事件数小于限制，说明已经没有更多事件
		if len(events) < limit {
			break
		}
	}

	return processedEvents, nil
}

// rebuildUserInbox 重建用户的收件箱
func (s *FeedService) rebuildUserInbox(ctx context.Context, userID int64, since time.Time, excludeEvents map[int64]bool) error {
	// 1. 获取用户关注的人
	followees, err := s.followRepo.GetFolloweeList(ctx, userID, 0, 1000) // 假设最多关注1000人
	if err != nil {
		return err
	}

	// 获取所有大V用户 (粉丝数超过阈值的用户)
	var bigVIDs []int64
	for _, followee := range followees {
		stats, err := s.followRepo.GetStatistics(ctx, followee.Followee)
		if err != nil {
			continue
		}

		// 是否为大V (大V通过拉模型获取)
		if stats.Followers > s.bigVThreshold {
			bigVIDs = append(bigVIDs, followee.Followee)
			continue
		}

		// 2. 为每个非大V的关注者获取自定起始时间以来的所有事件
		offset := 0
		limit := 100

		for {
			// 获取该关注者的Feed事件
			events, err := s.feedRepo.GetFeedEventsSince(ctx, since, offset, limit)
			if err != nil {
				// 记录错误但继续处理下一个关注者
				fmt.Printf("获取用户 %d 的Feed事件失败: %v\n", followee.Followee, err)
				break
			}

			// 如果没有更多事件，处理下一个关注者
			if len(events) == 0 {
				break
			}

			// 处理事件
			for _, event := range events {
				// 跳过已经处理过的事件 (避免重复)
				if excludeEvents[event.ID] {
					continue
				}

				// 只处理该关注者发起的事件
				if event.UserID != followee.Followee {
					continue
				}

				// 根据事件类型生成FeedItem并添加到收件箱
				feedItem, err := s.createFeedItemFromEvent(event)
				if err != nil {
					// 记录错误但继续处理
					fmt.Printf("从事件创建Feed项失败 (ID:%d): %v\n", event.ID, err)
					continue
				}

				// 添加到收件箱
				if err := s.feedRepo.AddToInbox(ctx, userID, feedItem); err != nil {
					// 记录错误但继续处理
					fmt.Printf("添加到收件箱失败 (ID:%d): %v\n", event.ID, err)
				}
			}

			// 更新偏移量
			offset += len(events)

			// 如果获取的事件数小于限制，说明已经没有更多事件
			if len(events) < limit {
				break
			}
		}
	}

	// 3. 对于大V用户，我们只需获取他们最近的10条Feed项即可 (拉模型)
	for _, bigVID := range bigVIDs {
		items, err := s.feedRepo.GetOutboxForUser(ctx, bigVID, 0, 10)
		if err != nil {
			// 记录错误但继续处理
			fmt.Printf("获取大V %d 的发件箱失败: %v\n", bigVID, err)
			continue
		}

		// 添加到当前用户的收件箱
		for _, item := range items {
			if err := s.feedRepo.AddToInbox(ctx, userID, item); err != nil {
				// 记录错误但继续处理
				fmt.Printf("添加大V项到收件箱失败: %v\n", err)
			}
		}
	}

	return nil
}

// createFeedItemFromEvent 从事件创建Feed项
func (s *FeedService) createFeedItemFromEvent(event domain.FeedEvent) (domain.UserFeedItem, error) {
	var itemType string
	var itemID int64

	switch event.EventType {
	case "article_published":
		itemType = "article"
		if id, ok := event.Content["article_id"].(float64); ok {
			itemID = int64(id)
		} else {
			return domain.UserFeedItem{}, fmt.Errorf("无效的文章ID")
		}

	case "article_liked":
		itemType = "like"
		if id, ok := event.Content["article_id"].(float64); ok {
			itemID = int64(id)
		} else {
			return domain.UserFeedItem{}, fmt.Errorf("无效的文章ID")
		}

	case "user_followed":
		itemType = "follow"
		if id, ok := event.Content["followee_id"].(float64); ok {
			itemID = int64(id)
		} else {
			return domain.UserFeedItem{}, fmt.Errorf("无效的被关注者ID")
		}

	case "article_commented":
		itemType = "comment"
		if id, ok := event.Content["article_id"].(float64); ok {
			itemID = int64(id)
		} else {
			return domain.UserFeedItem{}, fmt.Errorf("无效的文章ID")
		}

	case "article_collected":
		itemType = "collect"
		if id, ok := event.Content["article_id"].(float64); ok {
			itemID = int64(id)
		} else {
			return domain.UserFeedItem{}, fmt.Errorf("无效的文章ID")
		}

	default:
		return domain.UserFeedItem{}, fmt.Errorf("不支持的事件类型: %s", event.EventType)
	}

	return domain.NewUserFeedItem(itemType, itemID, event.UserID, event.Ctime, event.Content)
}

// CommentArticle 评论文章，产生 Feed 事件
func (s *FeedService) CommentArticle(ctx context.Context, userID, articleID, authorID, commentID int64, commentContent string) error {
	return s.feedProd.ProduceArticleCommentedEvent(ctx, userID, articleID, authorID, commentID, commentContent)
}

// PublishArticle 发布文章到 Feed
func (s *FeedService) PublishArticle(ctx context.Context, userID, articleID int64, title string) error {
	return s.feedProd.ProduceArticlePublishedEvent(ctx, userID, articleID, title)
}

// LikeArticle 点赞文章，产生 Feed 事件
func (s *FeedService) LikeArticle(ctx context.Context, userID, articleID, authorID int64) error {
	return s.feedProd.ProduceArticleLikedEvent(ctx, userID, articleID, authorID)
}

// FollowUser 关注用户，产生 Feed 事件
func (s *FeedService) FollowUser(ctx context.Context, followerID, followeeID int64) error {
	return s.feedProd.ProduceUserFollowedEvent(ctx, followerID, followeeID)
}

// CollectArticle 收藏文章，产生 Feed 事件
func (s *FeedService) CollectArticle(ctx context.Context, userID, articleID, authorID int64) error {
	return s.feedProd.ProduceArticleCollectedEvent(ctx, userID, articleID, authorID)
}
