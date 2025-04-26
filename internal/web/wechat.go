package web

import (
	"github.com/Fairy-nn/inspora/internal/service/oauth2/wechat"
	"github.com/gin-gonic/gin"
)

type WechatHandler struct {
	svc wechat.Service
}

func NewWechatHandler(svc wechat.Service) *WechatHandler {
	return &WechatHandler{svc: svc}

}

func (h *WechatHandler) RegisterRoutes(r *gin.Engine) {
	g := r.Group("/wechat")
	g.POST("/authurl", h.AuthURL)  // 微信登录
	g.Any("/callback", h.Callback) // 微信回调

}

func (h *WechatHandler) AuthURL(c *gin.Context) {
	// 获取微信授权链接
	url, err := h.svc.AuthURL(c)
	if err != nil {
		c.JSON(500, gin.H{"error": "获取微信授权链接失败"})
		return
	}
	c.JSON(200, gin.H{"url": url})
}

func (h *WechatHandler) Callback(c *gin.Context) {
}
