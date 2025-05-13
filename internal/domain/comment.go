package domain

type Comment struct {
	ID       int64      `json:"id"`
	Content  string     `json:"content"`
	UserID   int64      `json:"user_id"`
	UserName string     `json:"user_name"`
	Biz      string     `json:"biz"`
	BizID    int64      `json:"biz_id"`             // 业务ID
	ParentID int64      `json:"parent_id"`          // 父评论ID
	RootID   int64      `json:"root_id"`            // 根评论ID
	Ctime    int64      `json:"ctime"`              // 创建时间
	Children []*Comment `json:"children,omitempty"` // 子评论列表
}

// 判断评论是否是根评论
func (c *Comment) IsRootComment() bool {
	return c.ParentID <= 0
}

// 评论状态
type CommentStatus uint8

const (
	CommentStatusNormal CommentStatus = iota + 1 // 正常状态
	// 后面删除评论的时候，将状态改为删除状态，软删除
	CommentStatusDeleted // 已删除状态
)
