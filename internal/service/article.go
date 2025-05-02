package service

import (
	"context"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/repository"
)

// ArticleServiceInterface 文章服务接口
type ArticleServiceInterface interface {
	Save(ctx context.Context, article domain.Article) (int64, error)
	Publish(ctx context.Context, article domain.Article) (int64, error)
	Withdraw(ctx context.Context, article domain.Article) error
	List(ctx context.Context, userID int64, limit int, offset int) ([]domain.Article, error)
	FindById(ctx context.Context, id, uid int64) (domain.Article, error)
	FindPublicArticleById(ctx context.Context, id int64) (domain.Article, error)
}

// ArticleService 文章服务实现
type ArticleService struct {
	repo repository.ArticleRepository
}

// NewArticleService 创建文章服务
func NewArticleService(repo repository.ArticleRepository) ArticleServiceInterface {
	return &ArticleService{
		repo: repo,
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
func (a *ArticleService) FindPublicArticleById(ctx context.Context, id int64) (domain.Article, error) {
	return a.repo.FindPublicArticleById(ctx, id)
}