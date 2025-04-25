//go:build wireinject

package main

import (
	"github.com/Fairy-nn/inspora/internal/repository"
	"github.com/Fairy-nn/inspora/internal/repository/cache"
	"github.com/Fairy-nn/inspora/internal/repository/dao"
	"github.com/Fairy-nn/inspora/internal/service"
	"github.com/Fairy-nn/inspora/internal/web"
	"github.com/Fairy-nn/inspora/ioc"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
)

func InitInspora() *gin.Engine {
	wire.Build(
		ioc.InitDB,
		ioc.InitCache,
		ioc.InitSMS,
		ioc.InitGin,
		ioc.InitMiddlewares,

		cache.NewUserCacheV1,
		dao.NewUserDAO,
		repository.NewUserRepository,
		service.NewUserService,
		cache.NewCodeCache,
		repository.NewCodeRepository,
		service.NewCodeService,
		web.NewUserHandler,
	)

	return new(gin.Engine)
}
