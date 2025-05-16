package service

import (
	"context"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/pkg/elasticsearch"
)

// SearchService 搜索服务接口
type SearchService interface {
	// SearchUsers 搜索用户
	SearchUsers(ctx context.Context, query string, page, pageSize int) ([]domain.User, int64, error)
	// SearchArticles 搜索文章
	SearchArticles(ctx context.Context, query string, page, pageSize int) ([]elasticsearch.ArticleSearchResult, int64, error)
	// SearchArticlesByAuthor 按作者搜索文章
	SearchArticlesByAuthor(ctx context.Context, query string, authorID int64, page, pageSize int) ([]elasticsearch.ArticleSearchResult, int64, error)
	// IndexUser 索引用户
	IndexUser(ctx context.Context, user domain.User) error
	// IndexArticle 索引文章
	IndexArticle(ctx context.Context, article domain.Article) error
	// DeleteUserIndex 删除用户索引
	DeleteUserIndex(ctx context.Context, userID int64) error
	// DeleteArticleIndex 删除文章索引
	DeleteArticleIndex(ctx context.Context, articleID int64) error
}

type searchService struct {
	userSearchService    *elasticsearch.UserSearchService
	articleSearchService *elasticsearch.ArticleSearchService
}

// NewSearchService 创建搜索服务
func NewSearchService(
	userSearchService *elasticsearch.UserSearchService,
	articleSearchService *elasticsearch.ArticleSearchService,
) SearchService {
	return &searchService{
		userSearchService:    userSearchService,
		articleSearchService: articleSearchService,
	}
}

// SearchUsers 搜索用户
func (s *searchService) SearchUsers(ctx context.Context, query string, page, pageSize int) ([]domain.User, int64, error) {
	// 计算当前页的起始位置（用于数据库查询的offset）
	from := (page - 1) * pageSize
	// 调用用户搜索服务的Search方法进行搜索
	result, err := s.userSearchService.Search(ctx, query, from, pageSize)
	if err != nil {
		return nil, 0, err
	}
	// 处理搜索结果
	return s.userSearchService.ProcessSearchResult(result)
}

// SearchArticles 搜索文章
func (s *searchService) SearchArticles(ctx context.Context, query string, page, pageSize int) ([]elasticsearch.ArticleSearchResult, int64, error) {
	from := (page - 1) * pageSize
	result, err := s.articleSearchService.Search(ctx, query, from, pageSize)
	if err != nil {
		return nil, 0, err
	}

	return s.articleSearchService.ProcessSearchResult(result)
}

// SearchArticlesByAuthor 按作者搜索文章
func (s *searchService) SearchArticlesByAuthor(ctx context.Context, query string, authorID int64, page, pageSize int) ([]elasticsearch.ArticleSearchResult, int64, error) {
	from := (page - 1) * pageSize
	result, err := s.articleSearchService.SearchByAuthor(ctx, query, authorID, from, pageSize)
	if err != nil {
		return nil, 0, err
	}

	return s.articleSearchService.ProcessSearchResult(result)
}

// IndexUser 索引用户
func (s *searchService) IndexUser(ctx context.Context, user domain.User) error {
	return s.userSearchService.IndexUser(ctx, user)
}

// IndexArticle 索引文章
func (s *searchService) IndexArticle(ctx context.Context, article domain.Article) error {
	return s.articleSearchService.IndexArticle(ctx, article)
}

// DeleteUserIndex 删除用户索引
func (s *searchService) DeleteUserIndex(ctx context.Context, userID int64) error {
	return s.userSearchService.DeleteUser(ctx, userID)
}

// DeleteArticleIndex 删除文章索引
func (s *searchService) DeleteArticleIndex(ctx context.Context, articleID int64) error {
	return s.articleSearchService.DeleteArticle(ctx, articleID)
}
