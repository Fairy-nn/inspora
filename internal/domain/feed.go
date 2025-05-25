package domain

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

/*
三个结构体构成了 Feed 流的三级数据模型：
FeedEvent：原始事件记录，作为数据源，保持灵活性（Content 为 map）。
UserFeedItem：中间存储格式，将事件转换为用户 Feed 项，使用 JSON 字符串存储内容。
RenderedFeedItem：前端渲染格式，包含所有展示所需的字段（如用户名、标题、链接）。
*/

// FeedEvent 表示一个可能会产生 Feed 的事件
// 该结构体作为原始事件的载体，记录事件的基本信息
type FeedEvent struct {
	// 事件唯一标识
	ID int64 `json:"id"`
	// 事件发起者ID
	UserID int64 `json:"user_id"`
	// 事件类型，如 "article_published", "user_followed"
	EventType string `json:"event_type"`
	// 事件具体内容，结构根据事件类型变化
	Content map[string]interface{} `json:"content"`
	// 事件发生时间
	Ctime time.Time `json:"created_at"`
}

// UserFeedItem 表示用户收件箱中的一个 Feed 项
// 该结构体用于存储推送至用户 Feed 流的内容摘要
type UserFeedItem struct {
	// 复合ID，格式为 "类型:ID"（如 "article:123"）
	ItemID string `json:"item_id"`
	// 内容类型，如 "article", "follow"
	ItemType string `json:"item_type"`
	// 动作发起者ID（对应 FeedEvent 中的 UserID）
	ActorID int64 `json:"actor_id"`
	// 用于 Feed 排序的时间戳
	Timestamp time.Time `json:"timestamp"`
	// JSON 格式的内容摘要，存储结构化数据
	Content string `json:"content"`
}

func NewUserFeedItem(itemType string, itemID int64, actorID int64, timestamp time.Time, content map[string]interface{}) (UserFeedItem, error) {
	// 将内容详情序列化为 JSON 字符串
	contentJSON, err := json.Marshal(content)
	if err != nil {
		return UserFeedItem{}, fmt.Errorf("failed to marshal content: %w", err)
	}

	// 构建复合ID（格式: "类型:ID"）
	return UserFeedItem{
		ItemID:    itemType + ":" + strconv.FormatInt(itemID, 10),
		ItemType:  itemType,
		ActorID:   actorID,
		Timestamp: timestamp,
		Content:   string(contentJSON),
	}, nil
}

// RenderedFeedItem 表示渲染到前端的 Feed 项
// 该结构体是最终展示给用户的 Feed 格式，包含完整的渲染信息
type RenderedFeedItem struct {
	// 前端展示的唯一标识
	ID string `json:"id"`
	// 动作发起者名称（如用户名）
	ActorName string `json:"actor_name"`
	// 动作发起者ID
	ActorID int64 `json:"actor_id"`
	// 动作描述（如 "published", "followed"）
	Verb string `json:"verb"`
	// 动作对象类型（如 "article", "user"）
	Object string `json:"object"`
	// 动作对象ID
	ObjectID int64 `json:"object_id"`
	// 标题（如文章标题）
	Title string `json:"title"`
	// 内容摘要（如 "张三发布了文章《Go语言入门》"）
	Summary string `json:"summary"`
	// 点击跳转链接
	Link string `json:"link"`
	// 事件时间（用于展示和排序）
	Timestamp time.Time `json:"timestamp"`
}
