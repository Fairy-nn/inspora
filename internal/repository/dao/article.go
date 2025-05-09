package dao

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ArticleDaoInterface interface {
	Insert(ctx context.Context, article *Article) (int64, error)
	Update(ctx context.Context, article *Article) error
	Sync(ctx context.Context, article *Article) (int64, error)
	Upsert(ctx context.Context, article PublishArticle) error
	SyncStatus(ctx context.Context, articleID, authorID int64, status uint8) error
	FindByAuthor(ctx context.Context, authorID int64, offset, limit int) ([]Article, error)
	FindById(ctx context.Context, id, uid int64) (Article, error)
	FindPublicArticleById(ctx context.Context, id int64) (PublishArticle, error)
	ListPublic(ctx context.Context, startTime time.Time, offset, limit int) ([]Article, error)
}

// 这是制作库的数据库表结构
type Article struct {
	ID       int64  `gorm:"primaryKey,autoIncrement" json:"id"` // 文章ID
	Title    string `gorm:"type:varchar(1024)" json:"title"`    // 文章标题
	Content  string `gorm:"type:BLOB" json:"content"`           // 文章内容
	AuthorID int64  `gorm:"index:aid_ctime" json:"author_id"`   // 作者ID
	Ctime    int64  `gorm:"index:aid_ctime" json:"ctime"`       // 创建时间
	Utime    int64  `json:"utime"`
	Status   uint8  `json:"status"` // 文章状态
}

type ArticleGORMDAO struct {
	db *gorm.DB
}

func NewArticleDAO(db *gorm.DB) ArticleDaoInterface {
	return &ArticleGORMDAO{
		db: db,
	}
}

// 这个代表线上表（同库不同表）
type PublishArticle struct {
	Article
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
			"Status":  uint8(article.Status), // 文章状态
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

// Sync 同步文章
func (a *ArticleGORMDAO) Sync(ctx context.Context, article *Article) (int64, error) {
	var id = article.ID // 文章ID

	// Transaction 开启一个事务
	// 确保保存到线上库和制作库同时成功或失败
	// 闭包，GORM帮助我们管理事务的开始和提交，包括回滚、错误处理等
	var err error
	err = a.db.Transaction(func(tx *gorm.DB) error {
		// 先操作制作库，再操作线上库
		// txDao 执行的所有数据库操作都会在当前事务的上下文中进行
		txDao := NewArticleDAO(tx)

		if id > 0 {
			// 如果文章ID大于0，则更新文章
			err = txDao.Update(ctx, article)
		} else {
			// 创建新文章
			id, err = txDao.Insert(ctx, article)
		}

		if err != nil { //直接返回该错误，GORM 会自动回滚事务
			return err
		}

		// 同步到线上库
		err = txDao.Upsert(ctx, PublishArticle{Article: *article})
		return err
	})
	return id, err
}

// Upsert 插入或更新文章
func (a *ArticleGORMDAO) Upsert(ctx context.Context, article PublishArticle) error {
	now := time.Now().UnixMilli()
	article.Ctime = now
	article.Utime = now

	// 使用 GORM 的 Clauses 方法来执行 UPSERT 操作
	res := a.db.Clauses(clause.OnConflict{
		DoUpdates: clause.Assignments(map[string]interface{}{
			"title":   article.Title,
			"content": article.Content,
			"utime":   article.Utime,
			"status":  article.Status,
		}),
	}).Create(&article)
	// MYSQL最终的语句是 INSERT INTO article (title, content, ctime, updated_at) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE title = ?, content = ?, updated_at = ?

	return res.Error
}

// SyncStatus 同步文章状态
func (a *ArticleGORMDAO) SyncStatus(ctx context.Context, articleID, authorID int64, status uint8) error {
	now := time.Now().UnixMilli() // 获取当前时间戳

	// 开启一个事务
	return a.db.Transaction(func(tx *gorm.DB) error {
		// 更新Article表中的文章信息
		res := tx.WithContext(ctx).Model(&Article{}).Where("id = ? AND author_id = ?", articleID, authorID).Updates(
			map[string]any{
				"status": status,
				"utime":  now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("没有更新,article id: %d,author id :%d", articleID, authorID)
		}

		// 更新线上表中的文章信息
		return tx.WithContext(ctx).Model(&PublishArticle{}).Where("id = ? AND author_id = ?", articleID, authorID).Updates(
			map[string]any{
				"status": status,
				"utime":  now,
			}).Error
	})
}

// FindByAuthor 根据作者ID查找文章
func (a *ArticleGORMDAO) FindByAuthor(ctx context.Context, authorID int64, offset, limit int) ([]Article, error) {
	var articles []Article
	res := a.db.WithContext(ctx).Where("author_id = ?", authorID).Offset(offset).Limit(limit).Order("utime DESC").Find(&articles)
	if res.Error != nil {
		return nil, res.Error
	}

	return articles, nil
}

// FindById 根据文章ID查找文章
func (a *ArticleGORMDAO) FindById(ctx context.Context, id, uid int64) (Article, error) {
	var article Article
	res := a.db.WithContext(ctx).Where("id = ? AND author_id = ?", id, uid).First(&article)
	if res.Error != nil {
		return Article{}, res.Error
	}
	return article, nil
}

// FindPublicArticleById 根据文章ID查找公开文章
func (a *ArticleGORMDAO) FindPublicArticleById(ctx context.Context, id int64) (PublishArticle, error) {
	var pub PublishArticle
	err := a.db.WithContext(ctx).Where("id = ?", id).First(&pub).Error
	return pub, err
}

// ListPublic 根据时间获取文章列表
func (a *ArticleGORMDAO) ListPublic(ctx context.Context, startTime time.Time, offset, limit int) ([]Article, error) {
	var result []Article
	err := a.db.WithContext(ctx).Where("utime < ?", startTime.UnixMilli()).Order("utime DESC").Offset(offset).Limit(limit).Find(&result)
	return result, err.Error
}
