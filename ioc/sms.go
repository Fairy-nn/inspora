package ioc

import (
	"github.com/Fairy-nn/inspora/internal/service/sms"
	"github.com/Fairy-nn/inspora/internal/service/sms/memory"
)

func InitSMS() sms.Service {
	service := memory.NewMemorySMSService()
	return service
}
