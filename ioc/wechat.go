package ioc

import (
	"os"

	"github.com/Fairy-nn/inspora/internal/service/oauth2/wechat"
)

func InitOAuth2WechatService() wechat.Service {
	appId, ok := os.LookupEnv("WECHAT_APP_ID") // 获取微信应用ID
	if !ok {
		panic("WECHAT_APP_ID not set")
	}
	
	return wechat.NewService(appId)

}