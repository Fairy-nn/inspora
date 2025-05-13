package dao

import (
	"context"

	"gorm.io/gorm"
)

type Comment struct {
	ID      int64  `gorm:"primaryKey,autoIncrement" json:"id"` // 评论ID
	Content string `gorm:"type:text;not null" json:"content"`  // 评论内容
	// 在UserID上添加索引，方便查询
	UserID   int64  `gorm:"index:user_id;not null" json:"user_id"` // 用户ID
	UserName string `gorm:"type:varchar(100);not null"`            // 用户名称
	// 联合索引 biz+biz_id
	Biz   string `gorm:"type:varchar(20);not null;index:idx_biz_biz_id"` // 业务类型
	BizID int64  `gorm:"not null;index:idx_biz_biz_id"`                  // 业务ID
	// 在ParentID上添加索引
	ParentID int64 `gorm:"index;default:0"` // 父评论ID
	// 在RootID上添加索引
	RootID int64 `gorm:"index;default:0"` // 根评论ID
	Status uint8 `gorm:"default:1"`       // 评论状态
	Ctime  int64 `gorm:"autoCreateTime"`  // 创建时间
}

type CommentDAO interface {
	// Insert 插入评论
	Insert(ctx context.Context, comment Comment) (int64, error)
	// GetByID 根据ID获取评论
	GetByID(ctx context.Context, id int64) (Comment, error)
	// GetRootComments 获取根评论列表
	GetRootComments(ctx context.Context, biz string, bizID int64, minID int64, limit int) ([]Comment, error)
	// GetChildrenComments 获取子评论列表
	GetChildrenComments(ctx context.Context, parentID int64, minID int64, limit int) ([]Comment, error)
	// CountChildrenComments 统计子评论数量
	CountChildrenComments(ctx context.Context, parentID int64) (int64, error)
	// DeleteByID 删除评论及其所有子评论
	DeleteByID(ctx context.Context, id int64) error
	// GetHotComments 获取热门评论（按子评论数量降序）
	GetHotComments(ctx context.Context, biz string, bizID int64, limit int) ([]Comment, error)
	// GetUserById 获取用户信息
	GetUserById(ctx context.Context, userID int64) (User, error)
}

type CommentGORMDAO struct {
	db *gorm.DB
}

func NewCommentDAO(db *gorm.DB) CommentDAO {
	return &CommentGORMDAO{
		db: db,
	}
}

// Insert 插入评论
func (c *CommentGORMDAO) Insert(ctx context.Context, comment Comment) (int64, error) {
	// 如果是子评论，设置根评论ID
	if comment.ParentID > 0 {
		var parent Comment
		if err := c.db.WithContext(ctx).Where("id = ?", comment.ParentID).First(&parent).Error; err != nil {
			return 0, err //没有在数据库中找到父评论
		}
		if parent.RootID > 0 {
			// 如果父评论已经有根评论ID，直接使用父评论的根评论ID
			comment.RootID = parent.RootID
		} else {
			// 如果父评论没有根评论ID，设置根评论ID为父评论ID
			comment.RootID = parent.ID
		}
	}

	err := c.db.WithContext(ctx).Create(&comment).Error
	return comment.ID, err
}

// GetByID 根据评论ID获取评论
func (c *CommentGORMDAO) GetByID(ctx context.Context, id int64) (Comment, error) {
	var comment Comment
	err := c.db.WithContext(ctx).Where("id = ?", id).First(&comment).Error
	return comment, err
}

/**
OFFSET + LIMIT 的问题：
当偏移量（OFFSET）非常大时（例如 OFFSET 100000），数据库需要遍历并跳过前面的所有记录，即使这些记录最终不会被返回。这会导致查询性能急剧下降，时间复杂度为 O(N)。
在分页过程中，如果有新数据插入或旧数据删除，可能导致同一页数据重复或漏读
游标分页的优势：
通过 id < minID 直接定位到特定位置，无需遍历中间数据。查询时间复杂度为 O(1)
基于固定的 id 进行分页，即使数据发生变化，也不会影响已获取的页面。每个分页请求都是独立且稳定的。
**/
// GetRootComments 获取根评论列表
func (c *CommentGORMDAO) GetRootComments(ctx context.Context, biz string, bizID int64, minID int64, limit int) ([]Comment, error) {
	var comments []Comment
	query := c.db.WithContext(ctx).Where("biz = ? AND biz_id = ? AND parent_id <= 0", biz, bizID)
	// 使用 minID 作为分页条件
	if minID > 0 {
		query = query.Where("id < ?", minID)
	}
	// 按照 id 降序排列，可以保证最新的评论在前面
	// 使用 Limit 限制返回的评论数量
	err := query.Order("id DESC").Limit(limit).Find(&comments).Error

	return comments, err
}

// GetChildrenComments 获取子评论列表
func (c *CommentGORMDAO) GetChildrenComments(ctx context.Context, parentID int64, minID int64, limit int) ([]Comment, error) {
	var comments []Comment
	// 找parentID对应的评论
	query := c.db.WithContext(ctx).Where("parent_id = ?", parentID)
	if minID > 0 {
		// 使用 minID 作为分页条件
		query = query.Where("id < ?", minID)
	}

	err := query.Order("id DESC").Limit(limit).Find(&comments).Error
	return comments, err
}

// CountChildrenComments 统计子评论数量
func (c *CommentGORMDAO) CountChildrenComments(ctx context.Context, parentID int64) (int64, error) {
	var count int64
	err := c.db.WithContext(ctx).Model(&Comment{}).Where("parent_id = ?", parentID).Count(&count).Error
	return count, err
}

// DeleteByID 删除评论及其所有子评论
func (c *CommentGORMDAO) DeleteByID(ctx context.Context, id int64) error {
	// 为了避免失败，这里使用事务，然后递归的删除子评论
	return c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 递归删除子评论
		return c.recursiveDelete(tx, id)
	})
}
func (c *CommentGORMDAO) recursiveDelete(tx *gorm.DB, id int64) error {
	var childIDS []int64
	if err := tx.Model(&Comment{}).Where("parent_id = ?", id).Pluck("id", &childIDS).Error; err != nil {
		return err
	}
	// 递归删除子评论
	for _, childID := range childIDS {
		if err := c.recursiveDelete(tx, childID); err != nil {
			return err
		}
	}
	// 删除当前评论
	return tx.Where("id = ?", id).Delete(&Comment{}).Error
}

// GetHotComments 获取热门评论（按子评论数量降序）
func (c *CommentGORMDAO) GetHotComments(ctx context.Context, biz string, bizID int64, limit int) ([]Comment, error) {
	// 热门评论的定义是：子评论数量最多的评论
	var comments []Comment
	subQuery := c.db.WithContext(ctx).Model(&Comment{}).
		Select("parent_id, COUNT(*) as child_count").
		Where("parent_id > 0").
		Group("parent_id")

	err := c.db.WithContext(ctx).
		Table("comments c").
		Joins("LEFT JOIN (?) AS cc ON c.id = cc.parent_id", subQuery).
		Where("c.biz = ? AND c.biz_id = ? AND c.parent_id <= 0", biz, bizID).
		Order("COALESCE(cc.child_count, 0) DESC, c.id DESC").
		Limit(limit).
		Find(&comments).Error

	return comments, err
}

// GetUserById 获取用户信息
func (c *CommentGORMDAO) GetUserById(ctx context.Context, userID int64) (User, error) {
	var user User
	err := c.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error
	return user, err
}