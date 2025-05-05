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

// func InitInspora() *gin.Engine {
func InitApp() *App {
	wire.Build(
		ioc.InitDB,
		ioc.InitCache,
		ioc.InitSMS,
		ioc.InitGin,
		ioc.InitMiddlewares,

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

		ioc.InitKafka,
		ioc.NewSyncProducer,
		ioc.NewSyncConsumer,
		//events.NewKafkaConsumer,
		events.NewKafkaProducer,
		events.NewInteractionBatchConsumer,
		wire.Struct(new(App), "*"), // 绑定 App 结构体
	)

	//return new(gin.Engine)
	return new(App)
}
