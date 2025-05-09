package job

import (
	"context"
	"time"

	"github.com/Fairy-nn/inspora/internal/service"
)

type RankingJob struct {
	svc     service.RankingServiceInterface
	timeout time.Duration
}

func NewRankingJob(svc service.RankingServiceInterface) *RankingJob {
	return &RankingJob{
		svc:     svc,
		timeout: time.Minute,
	}
}

func (r *RankingJob) Name() string {
	return "Ranking Job"
}

func (r *RankingJob) Run() error {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	startTime := time.Now()
	println("【定时任务】排行榜任务开始执行 - 时间:", startTime.Format("2006-01-02 15:04:05"))

	// 调用服务层的TopN方法 计算文章的排名
	err := r.svc.TopN(ctx)

	endTime := time.Now()
	duration := endTime.Sub(startTime)
	if err != nil {
		println("【定时任务】排行榜任务执行失败 -", err.Error(), "- 耗时:", duration.String())
		return err
	}

	println("【定时任务】排行榜任务执行成功 - 时间:", endTime.Format("2006-01-02 15:04:05"), "- 耗时:", duration.String())
	return nil
}
