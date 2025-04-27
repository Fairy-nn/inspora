package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type ArticleDaoInterface interface {
	Insert(ctx context.Context, article *Article) (int64, error)
}

// 这是制作库的数据库表结构
type Article struct {
	ID       int64  `gorm:"primaryKey,autoIncrement"`
	Title    string `gorm:"type:varchar(1024)"` // 文章标题
	Content  string `gorm:"type:BLOB"`          // 文章内容
	AuthorID int64  `gorm:"index"`              // 作者ID
	Ctime    int64
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

func (a *ArticleGORMDAO) Insert(ctx context.Context, article *Article) (int64, error) {
	now:=time.Now().UnixMilli()
	article.Ctime = now
	article.Utime = now
	err := a.db.WithContext(ctx).Create(&article).Error
	return article.ID, err
}
