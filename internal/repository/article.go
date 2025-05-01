package repository

import (
	"context"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/repository/dao"
)

type ArticleRepository interface {
	// Create 创建文章
	Create(ctx context.Context, article domain.Article) (int64, error)
	// Update 更新文章
	Update(ctx context.Context, article domain.Article) error
	// Sync 同步文章
	Sync(ctx context.Context, article domain.Article) (int64, error)
	// SyncStatus 同步文章状态
	SyncStatus(ctx context.Context, articleID, authorID int64, status domain.ArticleStatus) error
}

type CachedArticleRepository struct {
	dao dao.ArticleDaoInterface
}

func NewCachedArticleRepository(dao dao.ArticleDaoInterface) ArticleRepository {
	return &CachedArticleRepository{dao: dao}
}

// SyncStatus 同步文章状态
func (c *CachedArticleRepository) SyncStatus(ctx context.Context, articleID, authorID int64, status domain.ArticleStatus) error {
	return c.dao.SyncStatus(ctx, articleID, authorID, status.ToUint8())
}

// Create 创建文章
func (c *CachedArticleRepository) Create(ctx context.Context, article domain.Article) (int64, error) {
	return c.dao.Insert(ctx, &dao.Article{
		Title:    article.Title,
		Content:  article.Content,
		AuthorID: article.Author.ID,
		Status:   article.Status.ToUint8(),
	})
}

// Update 更新文章
func (c *CachedArticleRepository) Update(ctx context.Context, article domain.Article) error {
	return c.dao.Update(ctx, &dao.Article{
		ID:       article.ID,
		Title:    article.Title,
		Content:  article.Content,
		AuthorID: article.Author.ID,
		Status:   article.Status.ToUint8(),
	})
}

// Sync 同步文章
func (c *CachedArticleRepository) Sync(ctx context.Context, article domain.Article) (int64, error) {
	return c.dao.Sync(ctx, &dao.Article{
		ID:       article.ID,
		Title:    article.Title,
		Content:  article.Content,
		AuthorID: article.Author.ID,
		Status:   article.Status.ToUint8(),
	})
}
