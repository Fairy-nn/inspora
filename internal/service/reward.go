package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/repository"
)

type RewardServiceInterface interface {
	// 预打赏，生成二维码
	PreReward(ctx context.Context, r domain.Reward) (domain.CodeURL, error)
	// 打赏，rid: reward id, uid: user id
	GetReward(ctx context.Context, rid, uid int64) (domain.Reward, error)
	// 更新打赏状态
	UpdateReward(ctx context.Context, bizTradeNo string, status domain.RewardStatus) error
}

type WechatNativeRewardService struct {
	svc     NativePaymentService // 支付服务
	repo    repository.RewardRepositoryInterface
	userSvc UserServiceInterface // 用户服务
}

func NewWechatNativeRewardService(svc NativePaymentService, repo repository.RewardRepositoryInterface, userSvc UserServiceInterface) RewardServiceInterface {
	return &WechatNativeRewardService{svc: svc, repo: repo, userSvc: userSvc}
}

// PreReward 预打赏，生成二维码
func (w *WechatNativeRewardService) PreReward(ctx context.Context, r domain.Reward) (domain.CodeURL, error) {
	// 如果在缓存中查到，则直接返回
	code, err := w.repo.GetCachedCodeURL(ctx, r)
	if err == nil {
		return code, nil
	}
	// 初始化状态
	r.Status = domain.RewardStatusInit
	rid, err := w.repo.CreateReward(ctx, r)
	if err != nil {
		return domain.CodeURL{}, err
	}
	// 调用支付的prepay方法创建二维码
	codeURL, err := w.svc.Prepay(ctx, domain.Payment{
		Amt: domain.Amount{
			Currency: "CNY",
			Total:    r.Amt,
		},
		BizTradeNo:  fmt.Sprintf("reward-%d", rid),
		Description: fmt.Sprintf("reward-%s", r.Target.BizName),
		Status:      domain.PaymentStatusInit,
		TxnID:       "",
	})
	if err != nil {
		return domain.CodeURL{}, err
	}
	code = domain.CodeURL{
		URL: codeURL,
		Rid: rid,
	}
	// 缓存二维码URL
	err = w.repo.CacheCodeURL(ctx, code, r)
	if err != nil {
		fmt.Println("cache code url failed", err)
	}
	return code, nil
}

func (w *WechatNativeRewardService) GetReward(ctx context.Context, rid, uid int64) (domain.Reward, error) {
	r, err := w.repo.GetReward(ctx, rid)
	if err != nil {
		return domain.Reward{}, err
	}

	// 检查用户是否是打赏者，否则是非法操作
	if r.UserID != uid {
		return domain.Reward{}, errors.New("reward not found")
	}

	// 如果打赏已完成则返回
	if r.Completed() {
		return r, nil
	}

	// 可能 reward 没有收到通知，那么就去 payment 查询一次
	resp, err := w.svc.GetPayment(ctx, w.toBizTradeNo(rid))
	if err != nil {
		fmt.Println("get payment failed", err)
		return r, nil
	}

	// 更新状态
	switch resp.Status {
	case domain.PaymentStatusFailed:
		r.Status = domain.RewardStatusFailed
	case domain.PaymentStatusSuccess:
		r.Status = domain.RewardStatusPaid
	case domain.PaymentStatusInit:
		r.Status = domain.RewardStatusInit
	// 如果打赏发生了退款，则更新状态为失败（但是我暂时不处理退款）
	case domain.PaymentStatusRefund:
		r.Status = domain.RewardStatusFailed
	}

	err = w.repo.UpdateStatus(ctx, rid, r.Status)
	if err != nil {
		fmt.Println("update reward status failed", err)
		return r, nil
	}

	return r, nil
}

func (w *WechatNativeRewardService) UpdateReward(ctx context.Context, bizTradeNo string, status domain.RewardStatus) error {
	rid := w.toRid(bizTradeNo)
	err := w.repo.UpdateStatus(ctx, rid, status)
	if err != nil {
		return err
	}

	// 如果打赏成功，进行分账处理
	if status == domain.RewardStatusPaid {
		reward, err := w.repo.GetReward(ctx, rid)
		if err != nil {
			return err
		}

		// 计算平台抽成（10%）
		platformFee := reward.Amt / 10
		// 计算用户实际获得的金额（90%）
		userAmount := reward.Amt - platformFee

		// 将用户的收益加入到用户余额中
		err = w.userSvc.UpdateUserBalance(ctx, reward.Target.UserID, userAmount)
		if err != nil {
			// 这里可以记录日志，但不影响流程
			fmt.Println("update user balance failed:", err)
		}
	}

	return nil
}

// toRid 将 bizTradeNo 转换为 rid
func (w *WechatNativeRewardService) toRid(bizTradeNo string) int64 {
	ridStr := strings.Split(bizTradeNo, "-")
	val, _ := strconv.ParseInt(ridStr[1], 10, 64)
	return val
}

// toBizTradeNo 将 rid 转换为 bizTradeNo
func (s *WechatNativeRewardService) toBizTradeNo(rid int64) string {
	return fmt.Sprintf("reward-%d", rid)
}
