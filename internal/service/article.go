package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	events "github.com/Fairy-nn/inspora/internal/events/article"
	feedevents "github.com/Fairy-nn/inspora/internal/events/feed"
	"github.com/Fairy-nn/inspora/internal/repository"
)

// ArticleServiceInterface 文章服务接口
type ArticleServiceInterface interface {
	Save(ctx context.Context, article domain.Article) (int64, error)
	Publish(ctx context.Context, article domain.Article) (int64, error)
	Withdraw(ctx context.Context, article domain.Article) error
	List(ctx context.Context, userID int64, limit int, offset int) ([]domain.Article, error)
	FindById(ctx context.Context, id, uid int64) (domain.Article, error)
	FindPublicArticleById(ctx context.Context, id int64, uid int64) (domain.Article, error)
	ListPublic(ctx context.Context, startTime time.Time, offset, limit int) ([]domain.Article, error)
}

// ArticleService 文章服务实现
type ArticleService struct {
	repo      repository.ArticleRepository
	producer  events.Producer
	searchSvc SearchService
	feedProd  feedevents.Producer // Feed 事件生产者
}

// NewArticleService 创建文章服务
func NewArticleService(repo repository.ArticleRepository,
	producer events.Producer, searchSvc SearchService,
	feedProd feedevents.Producer) ArticleServiceInterface {
	return &ArticleService{
		repo:      repo,
		producer:  producer,
		searchSvc: searchSvc,
		feedProd:  feedProd,
	}
}

// Save 保存文章
func (a *ArticleService) Save(ctx context.Context, article domain.Article) (int64, error) {
	// 设置文章状态为草稿
	article.Status = domain.ArticleStatusDraft

	// 如果文章ID大于0，则更新文章，否则创建新文章
	if article.ID > 0 {
		err := a.repo.Update(ctx, article)
		if err != nil {
			return article.ID, err
		}

		// 使用事务或锁确保数据一致性
		return article.ID, a.updateArticleIndex(ctx, article.ID, article.Author.ID)

		// // 更新索引
		// if err == nil && a.searchSvc != nil {
		// 	// 找到文章的完整信息
		// 	fullArticle, findErr := a.repo.FindById(ctx, article.ID, article.Author.ID)
		// 	if findErr == nil {
		// 		_ = a.searchSvc.IndexArticle(ctx, fullArticle)
		// 	}
		// }
		// return article.ID, err
	}
	// 创建新文章
	id, err := a.repo.Create(ctx, article)
	if err != nil {
		return 0, err
	}

	// 使用事务或锁确保数据一致性
	return id, a.updateArticleIndex(ctx, id, article.Author.ID)

	// id, err := a.repo.Create(ctx, article)
	// if err == nil && a.searchSvc != nil && id> 0 {
	// 	fullArticle, findErr := a.repo.FindById(ctx, id, article.Author.ID)
	// 	if findErr == nil {
	// 		_ = a.searchSvc.IndexArticle(ctx, fullArticle)
	// 	}
	// }
	// return id, err
}

// Publish 发布文章
func (a *ArticleService) Publish(ctx context.Context, article domain.Article) (int64, error) {
	// 设置文章状态为已发布
	article.Status = domain.ArticleStatusPublished
	// 同步到数据库
	id, err := a.repo.Sync(ctx, article)
	if err != nil {
		return id, err
	}

	// 更新搜索索引
	err = a.updateArticleIndex(ctx, id, article.Author.ID)
	if err != nil {
		log.Println("Failed to update article index:", err)
	}
	// 发送文章发布feed事件
	if a.feedProd != nil {
		go func() {
			feedErr := a.feedProd.ProduceArticlePublishedEvent(ctx, article.Author.ID, id, article.Title)
			if feedErr != nil {
				fmt.Println("Failed to send article published feed event:", feedErr)
			}
		}()
	}

	return id, nil
}

// Withdraw 撤回文章
func (a *ArticleService) Withdraw(ctx context.Context, article domain.Article) error {
	// 把文章撤回了，这里设置成草稿状态
	// return a.repo.SyncStatus(ctx, article.ID, article.Author.ID, domain.ArticleStatusDraft)
	if err := a.repo.SyncStatus(ctx, article.ID, article.Author.ID, domain.ArticleStatusDraft); err != nil {
		return err
	}

	// 更新搜索索引
	return a.updateArticleIndex(ctx, article.ID, article.Author.ID)
}

// List 获取文章列表
func (a *ArticleService) List(ctx context.Context, userID int64, limit int, offset int) ([]domain.Article, error) {
	return a.repo.List(ctx, userID, limit, offset)
}

// FindById 根据ID获取文章
func (a *ArticleService) FindById(ctx context.Context, id, uid int64) (domain.Article, error) {
	return a.repo.FindById(ctx, id, uid)
}

// FindPublicArticleById 根据ID获取公开文章
func (a *ArticleService) FindPublicArticleById(ctx context.Context, id int64, uid int64) (domain.Article, error) {
	// return a.repo.FindPublicArticleById(ctx, id)
	article, err := a.repo.FindPublicArticleById(ctx, id)
	if err == nil {
		go func() {
			// 发送浏览事件到 Kafka
			err := a.producer.ProducerViewEvent(ctx, events.ViewEvent{
				Uid: uid,
				Aid: id,
			})
			if err != nil {
				// 处理错误
				fmt.Println("Error producing view event:", err)
			}
		}()
	}
	return article, err
}

// ListPublic 取出规定时间内的文章列表
func (a *ArticleService) ListPublic(ctx context.Context, startTime time.Time, offset, limit int) ([]domain.Article, error) {
	// 根据时间获取文章列表
	return a.repo.ListPublic(ctx, startTime, offset, limit)
}

// updateArticleIndex 确保获取最新的文章数据并更新索引
func (a *ArticleService) updateArticleIndex(ctx context.Context, articleID, authorID int64) error {
	if a.searchSvc == nil {
		return nil
	}

	// 获取最新的文章数据
	fullArticle, err := a.repo.FindById(ctx, articleID, authorID)
	if err != nil {
		return fmt.Errorf("failed to find article %d for indexing: %w", articleID, err)
	}

	// 更新索引，不忽略错误
	if err := a.searchSvc.IndexArticle(ctx, fullArticle); err != nil {
		return fmt.Errorf("failed to index article %d: %w", articleID, err)
	}

	return nil
}
