package ioc

import (
	"github.com/Fairy-nn/inspora/internal/service/oauth2/wechat"
	"github.com/spf13/viper"
)

func InitOAuth2WechatService() wechat.Service {
	type Config struct {
		AppId     string `yaml:"app_id"`
		AppSecret string `yaml:"app_secret"`
	}
	var cfg = Config{
		AppId:     "wx426b3015558a9601",
		AppSecret: "your_app_secret",
	}
	err := viper.UnmarshalKey("wechat", &cfg)
	if err != nil {
		panic(err)
	}
	return wechat.NewService(cfg.AppId)

	// appId, ok := os.LookupEnv("WECHAT_APP_ID") // 获取微信应用ID
	// if !ok {
	// 	panic("WECHAT_APP_ID not set")
	// }

	// return wechat.NewService(appId)
}
