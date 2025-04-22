package service

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/Fairy-nn/inspora/internal/repository"
	"github.com/Fairy-nn/inspora/internal/service/sms"
)

type CodeService struct {
	repo   *repository.CodeRepository
	smsSvc sms.Service
}

func NewCodeService(repo *repository.CodeRepository) *CodeService {
	return &CodeService{
		repo: repo,
	}
}

func (s *CodeService) Send(ctx context.Context, biz, phone, code string) error {
	// 生成验证码
	code = s.generateCode() // 生成验证码
	// 存储到redis
	err := s.repo.Store(ctx, biz, phone, code) // 存储验证码到缓存
	if err != nil {
		return fmt.Errorf("验证码存储失败: %w", err) // 返回错误
	}
	// 发送验证码
	err = s.smsSvc.Send(ctx, biz, []string{code}, phone) // 发送验证码
	if err != nil {                                      //redis 中有验证码，但是失败
		return fmt.Errorf("验证码发送失败: %w", err) // 返回错误
	}
	return nil // 返回nil表示成功
}

// 生成验证`码
func (s *CodeService) generateCode() string {
	code := rand.Intn(1000000)       // 生成一个6位数的随机验证码
	return fmt.Sprintf("%06d", code) // 格式化为6位数，不足前面补0
}
