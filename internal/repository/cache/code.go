package cache

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
)

var luaSetCode string // lua脚本，设置验证码
// var luaVerifyCode string // lua脚本，验证验证码

var (
	ErrCodeSentTomanyTimes = errors.New("验证码发送次数过多")
	ErrCodeTriesTooMany    = errors.New("验证码尝试次数过多")
)

// 用于与 Redis 进行交互
type CodeCache struct {
	client redis.Cmdable
}

func NewCodeCache(client redis.Cmdable) *CodeCache {
	return &CodeCache{
		client: client,
	}
}

func (c *CodeCache) Set(ctx context.Context, biz, phone, code string) error {
	//传入上下文 ctx、脚本内容、键列表 []string{biz, phone} 和验证码 code，并将执行结果转换为整数类型。
	res, err := c.client.Eval(ctx, luaSetCode, []string{biz, phone}, code).Int()
	if err != nil {
		return err
	}
	switch res {
	case 0:
		return nil // 成功
	case -1:
		return ErrCodeSentTomanyTimes // 发送次数过多
	default:
		return errors.New("系统错误")
	}

}
