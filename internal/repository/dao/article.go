package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type ArticleDaoInterface interface {
	Insert(ctx context.Context, article *Article) (int64, error)
	Update(ctx context.Context, article *Article) ( error)
}

// 这是制作库的数据库表结构
type Article struct {
	ID       int64  `gorm:"primaryKey,autoIncrement"`
	Title    string `gorm:"type:varchar(1024)"` // 文章标题
	Content  string `gorm:"type:BLOB"`          // 文章内容
	AuthorID int64  `gorm:"index:aid_ctime"`              // 作者ID
	Ctime    int64	`gorm:"index:aid_ctime"`              // 创建时间
	Utime    int64
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
	now:=time.Now().UnixMilli()
	article.Ctime = now
	article.Utime = now
	err := a.db.WithContext(ctx).Create(&article).Error
	return article.ID, err
}
// Update 更新文章
func (a *ArticleGORMDAO) Update(ctx context.Context, article *Article) (error) {
	now := time.Now().UnixMilli()
	article.Utime = now

	// 使用 GORM 的 Updates 方法来更新文章的字段
	err := a.db.WithContext(ctx).Model(article).
	Where("id = ?", article.ID).
	Updates(map[string]any{
		"Title":    article.Title,
		"Content":  article.Content,
		"utime":  article.Utime,
	}).Error

	return err
}