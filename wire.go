//go:build wireinject

package main

import (
	events "github.com/Fairy-nn/inspora/internal/events/article"
	"github.com/Fairy-nn/inspora/internal/repository"
	"github.com/Fairy-nn/inspora/internal/repository/cache"
	"github.com/Fairy-nn/inspora/internal/repository/dao"
	"github.com/Fairy-nn/inspora/internal/service"
	"github.com/Fairy-nn/inspora/internal/web"
	"github.com/Fairy-nn/inspora/ioc"
	"github.com/google/wire"
)

var commentServiceSet = wire.NewSet(
	dao.NewCommentDAO,
	cache.NewRedisCommentCache,
	repository.NewCachedCommentRepository,
	service.NewCommentService,
	web.NewCommentHandler,
)

var followServiceSet = wire.NewSet(
	dao.NewFollowRelationDAO,
	cache.NewRedisFollowCache,
	repository.NewFollowRepository,
	service.NewFollowService,
	web.NewFollowHandler,
)

var searchServiceSet = wire.NewSet(
    ioc.ElasticsearchSet,
    ioc.SearchInitializerSet,
    service.NewSearchService,
    web.NewSearchHandler,
)

func InitApp()  (*App, error)  {
	wire.Build(
		ioc.InitDB,
		ioc.InitCache,
		ioc.InitSMS,
		ioc.InitGin,
		ioc.InitMiddlewares,
		ioc.InitKafka,
		ioc.NewSyncProducer,
		ioc.NewSyncConsumer,

		web.NewUserHandler,
		cache.NewUserCacheV1,
		dao.NewUserDAO,
		service.NewUserService,

		cache.NewCodeCache,
		repository.NewCodeRepository,
		service.NewCodeService,

		cache.NewRedisArticleCache,

		// ioc.InitOAuth2WechatService,
		// web.NewWechatHandler,
		repository.NewUserRepository,
		web.NewArticleHandler,
		service.NewArticleService,
		repository.NewCachedArticleRepository,
		dao.NewArticleDAO,

		repository.NewInteractionRepository,
		cache.NewRedisInteractionCache,
		dao.NewGormInteractionDAO,
		service.NewInteractionService,

		//events.NewKafkaConsumer,
		events.NewKafkaProducer,
		events.NewInteractionBatchConsumer,

		ioc.InitRankingRepository,
		service.NewBatchRankService,

		ioc.InitRankingJob,
		ioc.InitJobs,
		commentServiceSet,
		followServiceSet,
		searchServiceSet,

		wire.Struct(new(App), "*"), // 绑定 App 结构体
	)

	//return new(gin.Engine)
	return new(App), nil
}
