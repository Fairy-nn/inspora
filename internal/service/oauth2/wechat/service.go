package wechat

import (
	"context"
	"fmt"
	"net/url"

	"github.com/google/uuid"
)

type Service interface {
	AuthURL(ctx context.Context) (string, error) // 获取微信授权链接
}
type service struct {
	appId string // 微信应用ID
}

func NewService(appId string) Service {
	return &service{
		appId: appId,
	}
}

// NewService 创建微信服务
func (s *service) AuthURL(ctx context.Context) (string, error) {
	pattern_url := "https://open.weixin.qq.com/connect/qrconnect?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_login&state=%s#wechat_redirect"
	redirect_uri := url.PathEscape("https://stutuer.tech/oauth2/wechat/callback")
	state := uuid.New().String()
	return fmt.Sprintf(pattern_url, s.appId, redirect_uri, state), nil
}
