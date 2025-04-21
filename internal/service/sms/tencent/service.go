package tencent

import (
	"context"
	"fmt"

	"github.com/ecodeclub/ekit"
	sms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
)

type service struct {
	appId    *string
	signName *string
	client   *sms.Client
}

// NewService 创建一个新的腾讯云短信服务实例
func NewService(client *sms.Client, appId string, signName string) *service {
	return &service{
		appId:    &appId,
		signName: &signName,
		client:   client,
	}
}

// Send 发送短信
func (s *service) Send(ctx context.Context, biz string, args []string, numbers ...string) error {
	req := sms.NewSendSmsRequest() // 创建一个新的短信发送请求对象

	req.SmsSdkAppId = s.appId                          // 设置短信应用ID
	req.SignName = s.signName                          // 设置短信签名
	req.TemplateId = ekit.ToPtr[string](biz)           // 设置短信模板ID
	req.PhoneNumberSet = make([]*string, len(numbers)) // 设置接收短信的手机号码列表
	for i, number := range numbers {
		req.PhoneNumberSet[i] = &number
	}
	req.TemplateParamSet = make([]*string, len(args)) // 设置短信模板参数列表
	for i, arg := range args {
		req.TemplateParamSet[i] = &arg
	}

	resp, err := s.client.SendSms(req) // 发送短信请求

	if err != nil {
		return fmt.Errorf("腾讯短信服务发送失败 %w", err)
	}
	for _, status := range resp.Response.SendStatusSet {
		if status.Code == nil || *(status.Code) != "Ok" {
			return fmt.Errorf("发送短信失败 %s, %s ", *status.Code, *status.Message)
		}
	}
	return nil
}
