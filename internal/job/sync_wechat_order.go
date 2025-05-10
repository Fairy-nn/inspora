package job

import (
	"context"
	"fmt"
	"time"

	"github.com/Fairy-nn/inspora/internal/service"
)

type SyncWechatOrderJob struct {
	svc *service.NativePaymentService
}

func NewSyncWechatOrderJob(svc *service.NativePaymentService) *SyncWechatOrderJob {
	return &SyncWechatOrderJob{
		svc: svc,
	}
}

func (j *SyncWechatOrderJob) Name() string {
	return "sync_wechat_order_job"
}

func (j *SyncWechatOrderJob) Run() error {
	offset := 0
	limit := 100 // 每次查询100条数据,实现分页查询
	now := time.Now().Add(-time.Minute * 30) // 30分钟之前的时间

	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		// 查询创建时间在30分钟之前且支付状态为未支付的订单
		pmts, err := j.svc.FindExpiredPayment(ctx, offset, limit, now)
		cancel()

		if err != nil {
			return err
		}
		// 遍历查询到的订单
		for _, pm := range pmts {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			// 调用微信支付API同步订单信息
			err := j.svc.SyncWechatInfo(ctx, pm.BizTradeNo)
			if err != nil {
				fmt.Println("SyncWechatInfo error:", err)
			}
			cancel()
		}

		// 如果查询到的支付记录数量小于限制数量，说明没有更多数据了，可以退出循环
		if len(pmts) < limit {
			return nil
		}

		offset += len(pmts)
	}
}
