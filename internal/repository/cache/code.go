package cache

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

//go:embed lua/set_code.lua
var luaSetCode string // lua脚本，设置验证码
//go:embed lua/verify_code.lua
var luaVerifyCode string // lua脚本，验证验证码

var (
	ErrCodeSentTomanyTimes = errors.New("验证码发送次数过多")
	ErrCodeTriesTooMany    = errors.New("验证码尝试次数过多")
	ErrCodeNotExpired       = errors.New("验证码未过期，请一分钟后再试")
)

type CodeCacheInterface interface {
	Set(ctx context.Context, biz, phone, code string) error
	Verify(ctx context.Context, biz, phone, code string) (bool, error)
}

// 用于与 Redis 进行交互
type CodeCache struct {
	client redis.Cmdable
}

func NewCodeCache(client redis.Cmdable) CodeCacheInterface {
	return &CodeCache{
		client: client,
	}
}

// Set 方法用于设置验证码
func (c *CodeCache) Set(ctx context.Context, biz, phone, code string) error {
	//传入上下文 ctx、脚本内容、键列表 []string{biz, phone} 和验证码 code，并将执行结果转换为整数类型。
	res, err := c.client.Eval(ctx, luaSetCode, []string{fmt.Sprintf("code:%s:%s", biz, phone)}, code).Int()
	if err != nil {
		fmt.Printf("验证码存储失败: %v\n", err) // 打印错误信息
		return err
	}
	switch res {
	case 1:
		fmt.Println("验证码未过期，请一分钟后再试") // 打印错误信息
		return ErrCodeNotExpired // 验证码未过期，请一分钟后再试
	case 0:
		return nil // 成功
	case -1:
		fmt.Println("验证码发送次数过多")  // 打印错误信息
		return ErrCodeSentTomanyTimes // 发送次数过多
	default:
		return errors.New("系统错误")
	}
}

// Verify 方法用于验证验证码
func (c *CodeCache) Verify(ctx context.Context, biz, phone, code string) (bool, error) {
	res, err := c.client.Eval(ctx, luaVerifyCode, []string{fmt.Sprintf("code:%s:%s", biz, phone)}, code).Int()
	if err != nil {
		return false, err
	}
	switch res {
	case 0:
		return true, nil
	case -1:
		return false, ErrCodeTriesTooMany // 尝试次数过多
	case -2:
		return false, nil
	default:
		return false, errors.New("系统错误")
	}
}
