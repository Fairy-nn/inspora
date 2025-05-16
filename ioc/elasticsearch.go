package ioc

import (
	"context"
	"fmt"
	"log"

	"github.com/Fairy-nn/inspora/config"
	"github.com/Fairy-nn/inspora/pkg/elasticsearch"
	"github.com/google/wire"
	"github.com/spf13/viper"
)

// SearchInitializer 搜索初始化器接口，用于替代原来的Factory
type SearchInitializer interface {
	// InitializeIndices 初始化所有索引
	InitializeIndices(ctx context.Context) error
	// Close 关闭资源
	Close() error
}

// DefaultSearchInitializer 默认搜索初始化器实现
type DefaultSearchInitializer struct {
	UserSearch    *elasticsearch.UserSearchService
	ArticleSearch *elasticsearch.ArticleSearchService
}

// InitializeIndices 初始化所有索引
func (i *DefaultSearchInitializer) InitializeIndices(ctx context.Context) error {
	// 初始化用户索引
	if err := i.UserSearch.EnsureIndex(ctx); err != nil {
		return fmt.Errorf("failed to initialize user index: %w", err)
	}

	// 初始化文章索引
	if err := i.ArticleSearch.EnsureIndex(ctx); err != nil {
		return fmt.Errorf("failed to initialize article index: %w", err)
	}

	return nil
}

// Close 关闭搜索初始化器
func (i *DefaultSearchInitializer) Close() error {
	// 当前ES客户端没有提供显式关闭方法，未来版本可能增加
	return nil
}

// ProvideElasticSearchConfig 提供Elasticsearch配置
func ProvideElasticSearchConfig() *config.ElasticSearchConfig {
	var cfg config.ElasticSearchConfig
	if err := viper.UnmarshalKey("elasticsearch", &cfg); err != nil {
		log.Fatalf("failed to unmarshal elasticsearch config: %v", err)
	}
	return &cfg
}

// ProvideSearchInitializer 提供搜索初始化器
func ProvideSearchInitializer(userSearch *elasticsearch.UserSearchService, articleSearch *elasticsearch.ArticleSearchService) *DefaultSearchInitializer {
	return &DefaultSearchInitializer{
		UserSearch:    userSearch,
		ArticleSearch: articleSearch,
	}
}

// InitializeElasticsearchIndices 初始化Elasticsearch索引
func InitializeElasticsearchIndices(initializer SearchInitializer) {
	if err := initializer.InitializeIndices(context.Background()); err != nil {
		log.Fatalf("failed to initialize elasticsearch indices: %v", err)
	}
}

// ElasticsearchSet 定义Elasticsearch服务的依赖关系集合
var ElasticsearchSet = wire.NewSet(
	ProvideElasticSearchConfig,
	elasticsearch.NewClient,
	elasticsearch.NewBaseIndexService,
	elasticsearch.NewBaseSearchService,
	elasticsearch.NewUserSearchService,
	elasticsearch.NewArticleSearchService,
)

// SearchInitializerSet 定义搜索初始化器的依赖关系集合
var SearchInitializerSet = wire.NewSet(
	ProvideSearchInitializer,
	wire.Bind(new(SearchInitializer), new(*DefaultSearchInitializer)),
)
