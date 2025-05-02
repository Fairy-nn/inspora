package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/repository/cache"
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
	// List 获取文章列表
	List(ctx context.Context, userID int64, limit int, offset int) ([]domain.Article, error)
	// FindById 根据ID获取文章
	FindById(ctx context.Context, id, uid int64) (domain.Article, error)
}

type CachedArticleRepository struct {
	dao   dao.ArticleDaoInterface
	cache cache.ArticleCache // 文章缓存
}

func NewCachedArticleRepository(dao dao.ArticleDaoInterface, cache cache.ArticleCache) ArticleRepository {
	return &CachedArticleRepository{dao: dao, cache: cache}
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

// List 获取文章列表
func (c *CachedArticleRepository) List(ctx context.Context, userID int64, limit int, offset int) ([]domain.Article, error) {
	// 核心是先查缓存，再查询数据库
	// 只缓存了第一页
	if offset == 0 && limit <= 100 {
		// 直接从缓存中获取
		articles, err := c.cache.GetFirstPage(ctx, userID)
		if err != nil {
			return nil, err
		}

		// 异步进行预缓存操作，这里是预测用户会点击第一个进行访问
		if len(articles) > 0 {
			go func() {
				// 异步更新缓存
				c.preCache(ctx, articles)
			}()
		}
	}
	
	// 从数据库中获取文章列表
	res, err := c.dao.FindByAuthor(ctx, userID, offset, limit)
	if err != nil {
		return nil, err
	}

	articles := c.toDomainList(res)

	// 异步回写缓存
	go func() {
		// 将查询到的文章数据写入缓存
		err := c.cache.SetFirstPage(ctx, userID, articles)
		if err != nil {
			fmt.Println("回写缓存失败", err)
		}
		c.preCache(ctx, articles)
	}()

	return articles, nil
}

// 将数据库对象转换为领域对象
func (c *CachedArticleRepository) toDomainList(articles []dao.Article) []domain.Article {
	var result []domain.Article
	for _, a := range articles {
		result = append(result, domain.Article{
			ID:      a.ID,
			Title:   a.Title,
			Content: a.Content,
			Author:  domain.Author{ID: a.AuthorID},
			Status:  domain.ArticleStatus(a.Status),
			Ctime:   time.UnixMilli(a.Ctime),
			Utime:   time.UnixMilli(a.Utime),
		})
	}
	return result

}

// preCache 预缓存,而且缓存时间很短
func (c *CachedArticleRepository) preCache(ctx context.Context, articles []domain.Article) {
	if len(articles) > 0 && len(articles[0].Content) < 1024*1024 {
		// 预缓存第一篇文章
		err := c.cache.Set(ctx, articles[0], articles[0].Author.ID)
		if err != nil {
			fmt.Println("预缓存失败", err)
		}
	}
}

// FindById 根据ID获取文章
func (c *CachedArticleRepository) FindById(ctx context.Context, id, uid int64) (domain.Article, error) {
	article, err := c.dao.FindById(ctx, id, uid)
 	if err != nil {
 		return domain.Article{}, err
 	}
 	return domain.Article{
		ID:	  article.ID,
		Title:   article.Title,
		Content: article.Content,
		Author:  domain.Author{ID: article.AuthorID},
		Status:  domain.ArticleStatus(article.Status),
		Ctime:   time.UnixMilli(article.Ctime),
		Utime:   time.UnixMilli(article.Utime),
	}, nil
}