package service

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/Fairy-nn/inspora/internal/repository"
	"github.com/Fairy-nn/inspora/internal/service/sms"
)

type CodeServiceInterface interface {
	Send(ctx context.Context, biz, phone string) error
	Verify(ctx context.Context, biz, phone, code string) (bool, error)
}

type CodeService struct {
	repo   repository.CodeRepositoryInterface
	smsSvc sms.Service
}

func NewCodeService(repo repository.CodeRepositoryInterface, smsSvc sms.Service) CodeServiceInterface {
	return &CodeService{
		repo:   repo,
		smsSvc: smsSvc,
	}
}

func (s *CodeService) Send(ctx context.Context, biz, phone string) error {
	// 生成验证码
	code := s.generateCode() // 生成验证码
	// 存储到redis
	err := s.repo.Store(ctx, biz, phone, code)

	if err != nil {
		return err
	}
	// 发送验证码
	err = s.smsSvc.Send(ctx, biz, []string{code}, phone) // 发送验证码
	if err != nil {                                      //redis 中有验证码，但是失败
		return fmt.Errorf("验证码发送失败: %w", err) // 返回错误
	}
	return nil // 返回nil表示成功
}

// 生成验证码
func (s *CodeService) generateCode() string {
	code := rand.Intn(1000000)       // 生成一个6位数的随机验证码
	return fmt.Sprintf("%06d", code) // 格式化为6位数，不足前面补0
}

// Verify 验证验证码
func (s *CodeService) Verify(ctx context.Context, biz, phone, code string) (bool, error) {
	// 调用存储库的 Verify 方法验证验证码
	ok, err := s.repo.Verify(ctx, biz, phone, code) // 验证验证码
	if err != nil {                                 // 如果验证失败，返回错误
		return false, err // 返回错误
	}
	return ok, nil // 返回验证结果
}
