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
}

type CachedArticleRepository struct {
	dao dao.ArticleDaoInterface
}

func NewCachedArticleRepository(dao dao.ArticleDaoInterface) ArticleRepository {
	return &CachedArticleRepository{dao: dao}
}

// Create 创建文章
func (c *CachedArticleRepository) Create(ctx context.Context, article domain.Article) (int64, error) {
	return c.dao.Insert(ctx, &dao.Article{
		Title:    article.Title,
		Content:  article.Content,
		AuthorID: article.Author.ID,
	})
}

// Update 更新文章
func (c *CachedArticleRepository) Update(ctx context.Context, article domain.Article) error {
	return c.dao.Update(ctx, &dao.Article{
		ID:       article.ID,
		Title:    article.Title,
		Content:  article.Content,
		AuthorID: article.Author.ID,
	})
}
// Sync 同步文章
func (c *CachedArticleRepository) Sync(ctx context.Context, article domain.Article) (int64, error) {
	return c.dao.Sync(ctx, &dao.Article{
		ID:       article.ID,
		Title:    article.Title,
		Content:  article.Content,
		AuthorID: article.Author.ID,
	})
}