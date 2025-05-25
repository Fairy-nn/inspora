package repository

import (
	"context"
	"encoding/json"
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
	// FindPublicArticleById 根据ID获取公开文章
	FindPublicArticleById(ctx context.Context, id int64) (domain.Article, error)
	// ListPublic 获取公开文章列表
	ListPublic(ctx context.Context, startTime time.Time, offset, limit int) ([]domain.Article, error)
}

type CachedArticleRepository struct {
	dao      dao.ArticleDaoInterface
	cache    cache.ArticleCache      // 文章缓存
	userRepo UserRepositoryInterface // 用户仓库
}

func NewCachedArticleRepository(dao dao.ArticleDaoInterface, cache cache.ArticleCache, repo UserRepositoryInterface) ArticleRepository {
	return &CachedArticleRepository{dao: dao, cache: cache, userRepo: repo}
}

// SyncStatus 同步文章状态
func (c *CachedArticleRepository) SyncStatus(ctx context.Context, articleID, authorID int64, status domain.ArticleStatus) error {
	return c.dao.SyncStatus(ctx, articleID, authorID, status.ToUint8())
}

// Create 创建文章
func (c *CachedArticleRepository) Create(ctx context.Context, article domain.Article) (int64, error) {
	// 讲图片地址转换为字符串
	imageURLsJSON, err := json.Marshal(article.ImgUrls)
	if err != nil {
		return 0, fmt.Errorf("转换图片地址失败: %w", err)
	}
	// 删除缓存
	defer func() {
		err := c.cache.DelFirstPage(ctx, article.Author.ID)
		if err != nil {
			fmt.Println("删除缓存失败", err)
		}
	}()

	return c.dao.Insert(ctx, &dao.Article{
		Title:    article.Title,
		Content:  article.Content,
		AuthorID: article.Author.ID,
		Status:   article.Status.ToUint8(),
		ImgUrls:  string(imageURLsJSON),
	})
}

// Update 更新文章
func (c *CachedArticleRepository) Update(ctx context.Context, article domain.Article) error {
	// 讲图片地址转换为字符串
	imageURLsJSON, err := json.Marshal(article.ImgUrls)
	if err != nil {
		return fmt.Errorf("转换图片地址失败: %w", err)
	}
	defer func() {
		err := c.cache.DelFirstPage(ctx, article.Author.ID)
		if err != nil {
			fmt.Println("删除缓存失败", err)
		}

		err = c.cache.SetPub(ctx, article)
		if err != nil {
			fmt.Println("设置公共缓存失败", err)
		}
	}()

	return c.dao.Update(ctx, &dao.Article{
		ID:       article.ID,
		Title:    article.Title,
		Content:  article.Content,
		AuthorID: article.Author.ID,
		Status:   article.Status.ToUint8(),
		ImgUrls:  string(imageURLsJSON), // 图片地址
	})
}

// Sync 同步文章
func (c *CachedArticleRepository) Sync(ctx context.Context, article domain.Article) (int64, error) {
	id, err := c.dao.Sync(ctx, c.toEntity(article))

	// 文章发布成功后，删除缓存
	if err == nil {
		// 删除缓存
		err := c.cache.DelFirstPage(ctx, article.Author.ID)
		if err != nil {
			fmt.Println("删除缓存失败", err)
		}

		err = c.cache.SetPub(ctx, article) // 异步设置公共缓存
		if err != nil {
			fmt.Println("设置公共缓存失败", err)
		}
	}
	return id, err
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
		article := domain.Article{
			ID:      a.ID,
			Title:   a.Title,
			Content: a.Content,
			Author:  domain.Author{ID: a.AuthorID},
			Status:  domain.ArticleStatus(a.Status),
			Ctime:   time.UnixMilli(a.Ctime),
			Utime:   time.UnixMilli(a.Utime),
		}
		if a.ImgUrls != "" {
			// 解析图片地址
			var imgUrls []string
			err := json.Unmarshal([]byte(a.ImgUrls), &imgUrls)
			if err != nil {
				fmt.Println("解析图片地址失败", err)
			}
			article.ImgUrls = imgUrls
		}
		result = append(result, article)
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
	return c.toDomain(article), nil
}

// FindPublicArticleById 根据ID获取公开文章
func (c *CachedArticleRepository) FindPublicArticleById(ctx context.Context, id int64) (domain.Article, error) {
	// 先从缓存中获取文章
	cachedArticle, err := c.cache.GetPub(ctx, id)
	if err == nil && cachedArticle.ID > 0 {
		return cachedArticle, nil
	}

	// 如果缓存中没有，则从数据库中获取
	article, err := c.dao.FindPublicArticleById(ctx, id)
	if err != nil {
		return domain.Article{}, err
	}

	// 根据文章的作者ID，从用户仓库中获取作者信息
	user, err := c.userRepo.GetByID(ctx, article.AuthorID)
	if err != nil {
		return domain.Article{}, err
	}
	// 组合成domain.Article对象
	res := domain.Article{
		ID:      article.ID,
		Title:   article.Title,
		Content: article.Content,
		Author:  domain.Author{ID: user.ID, Name: user.Name},
		Status:  domain.ArticleStatus(article.Status),
		Ctime:   time.UnixMilli(article.Ctime),
		Utime:   time.UnixMilli(article.Utime),
	}

	// 开启一个异步将文章信息存入缓存
	go func() {
		err := c.cache.SetPub(ctx, res)
		if err != nil {
			fmt.Println("设置公共缓存失败", err)
		}
	}()

	return res, nil
}

// ListPublic 获取公开文章列表
func (c *CachedArticleRepository) ListPublic(ctx context.Context, startTime time.Time, offset, limit int) ([]domain.Article, error) {
	res, err := c.dao.ListPublic(ctx, startTime, offset, limit)
	if err != nil {
		return nil, err
	}
	return c.toDomainList(res), nil
}

func (c *CachedArticleRepository) toDomain(a dao.Article) domain.Article {
	article := domain.Article{
		ID:      a.ID,
		Title:   a.Title,
		Content: a.Content,
		Author:  domain.Author{ID: a.AuthorID},
		Status:  domain.ArticleStatus(a.Status),
		Ctime:   time.UnixMilli(a.Ctime),
		Utime:   time.UnixMilli(a.Utime),
	}

	// 解析图片地址
	if a.ImgUrls != "" {
		var imageURLs []string
		if err := json.Unmarshal([]byte(a.ImgUrls), &imageURLs); err == nil {
			article.ImgUrls = imageURLs
		}
	}

	return article
}

func (c *CachedArticleRepository) toEntity(a domain.Article) *dao.Article {
	// 将图片地址转换为字符串
	imageURLsJSON, _ := json.Marshal(a.ImgUrls)
	return &dao.Article{
		ID:       a.ID,
		Title:    a.Title,
		Content:  a.Content,
		AuthorID: a.Author.ID,
		Status:   a.Status.ToUint8(),
		ImgUrls:  string(imageURLsJSON),
	}
}
