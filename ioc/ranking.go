package ioc

import (
	"github.com/Fairy-nn/inspora/internal/job"
	"github.com/Fairy-nn/inspora/internal/repository"
	"github.com/Fairy-nn/inspora/internal/repository/cache"
	"github.com/Fairy-nn/inspora/internal/service"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
)

func InitRankingJob(svc service.RankingServiceInterface) *job.RankingJob {
	return job.NewRankingJob(svc)
}

// InitRankingRepository 创建排行榜仓库
func InitRankingRepository(cmdable redis.Cmdable) repository.RankingRepositoryInterface {
	redisCache := cache.NewRedisRankingCache(cmdable)
	localCache := cache.NewLocalRankingCache()
	return repository.NewCachedRankingRepository(redisCache, *localCache)
}

// 初始化定时任务，这里使用了robfig/cron库来实现定时任务
func InitJobs(rankingJob *job.RankingJob) *cron.Cron {
	expr := cron.New(cron.WithSeconds())
	// 每三分钟执行一次
	_, err := expr.AddJob("0 */3 * * * *", job.NewCornJobBuilder().Build(rankingJob))
	if err != nil {
		panic(err)
	}
	return expr
}
