package main

import (
	"fmt"
)

func main() {
	if err := InitViper(); err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	r := InitInspora()
	// v := ioc.InitMiddlewares()
	// db := ioc.InitDB()
	// userDaoInterface := dao.NewUserDAO(db)
	// cmdable := ioc.InitCache()
	// userCacheInterface := cache.NewUserCacheV1(cmdable)
	// userRepositoryInterface := repository.NewUserRepository(userDaoInterface, userCacheInterface)
	// userServiceInterface := service.NewUserService(userRepositoryInterface)
	// codeCacheInterface := cache.NewCodeCache(cmdable)
	// codeRepositoryInterface := repository.NewCodeRepository(codeCacheInterface)
	// smsService := ioc.InitSMS()
	// codeServiceInterface := service.NewCodeService(codeRepositoryInterface, smsService)
	// userHandler := web.NewUserHandler(userServiceInterface, codeServiceInterface)
	// articleDaoInterface := dao.NewArticleDAO(db)
	// articleRepository := repository.NewCachedArticleRepository(articleDaoInterface)
	// articleServiceInterface := service.NewArticleService(articleRepository)
	// articleHandler := web.NewArticleHandler(articleServiceInterface)
	// r := ioc.InitGin(v, userHandler, articleHandler)
	println("服务启动成功")
	fmt.Println(r)
	r.Run(":8080") // 启动服务器
}
