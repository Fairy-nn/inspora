package dao

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type ArticleDaoInterface interface {
	Insert(ctx context.Context, article *Article) (int64, error)
	Update(ctx context.Context, article *Article) error
}

// 这是制作库的数据库表结构
type Article struct {
	ID       int64  `gorm:"primaryKey,autoIncrement" json:"id"` // 文章ID
	Title    string `gorm:"type:varchar(1024)" json:"title"`    // 文章标题
	Content  string `gorm:"type:BLOB" json:"content"`           // 文章内容
	AuthorID int64  `gorm:"index:aid_ctime" json:"author_id"`   // 作者ID
	Ctime    int64  `gorm:"index:aid_ctime" json:"ctime"`       // 创建时间
	Utime    int64  `json:"utime"`
}

type ArticleGORMDAO struct {
	db *gorm.DB
}

func NewArticleDAO(db *gorm.DB) ArticleDaoInterface {
	return &ArticleGORMDAO{
		db: db,
	}
}

// Insert 插入文章
func (a *ArticleGORMDAO) Insert(ctx context.Context, article *Article) (int64, error) {
	now := time.Now().UnixMilli()
	article.Ctime = now
	article.Utime = now
	err := a.db.WithContext(ctx).Create(&article).Error
	return article.ID, err
}

// Update 更新文章
func (a *ArticleGORMDAO) Update(ctx context.Context, article *Article) error {
	now := time.Now().UnixMilli()
	article.Utime = now
	// 为了避免攻击者假冒用户修改其他用户的文章
	// 使用 GORM 的 Updates 方法来更新文章的字段
	res := a.db.WithContext(ctx).Model(article).
		Where("id = ? AND author_id = ?", article.ID, article.AuthorID).
		Updates(map[string]any{
			"Title":   article.Title,
			"Content": article.Content,
			"Utime":   article.Utime,
		})
	if res.Error != nil {
		return res.Error
	}

	// 如果更新的行数为 0，表示没有更新任何行
	if res.RowsAffected == 0 {
		fmt.Println("没有更新任何行")
		return fmt.Errorf("没有更新,article id: %d,author id :%d", article.ID, article.AuthorID)
	}

	return nil
}
