package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	events "github.com/Fairy-nn/inspora/internal/events/article"
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
	repo     repository.ArticleRepository
	producer events.Producer
}

// NewArticleService 创建文章服务
func NewArticleService(repo repository.ArticleRepository,
	producer events.Producer) ArticleServiceInterface {
	return &ArticleService{
		repo:     repo,
		producer: producer,
	}
}

// Save 保存文章
func (a *ArticleService) Save(ctx context.Context, article domain.Article) (int64, error) {
	// 设置文章状态为草稿
	article.Status = domain.ArticleStatusDraft

	// 如果文章ID大于0，则更新文章，否则创建新文章
	if article.ID > 0 {
		return article.ID, a.repo.Update(ctx, article)
	}

	return a.repo.Create(ctx, article)
}

// Publish 发布文章
func (a *ArticleService) Publish(ctx context.Context, article domain.Article) (int64, error) {
	// 设置文章状态为已发布
	article.Status = domain.ArticleStatusPublished

	return a.repo.Sync(ctx, article)
}

// Withdraw 撤回文章
func (a *ArticleService) Withdraw(ctx context.Context, article domain.Article) error {
	// 把文章撤回了，这里设置成草稿状态
	return a.repo.SyncStatus(ctx, article.ID, article.Author.ID, domain.ArticleStatusDraft)
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