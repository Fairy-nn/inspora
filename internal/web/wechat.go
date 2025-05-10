package web

import (
	"net/http"

	"github.com/Fairy-nn/inspora/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
)

// type WechatHandler struct {
// 	svc wechat.Service
// }

// func NewWechatHandler(svc wechat.Service) *WechatHandler {
// 	return &WechatHandler{svc: svc}

// }

func (h *WeChatPaymentHandler) RegisterRoutes(r *gin.Engine) {
	g := r.Group("/wechat")
	g.Any("/pay/callback", h.HandleNative) // 微信支付回调
	// g.POST("/authurl", h.AuthURL)          // 微信登录
	// g.Any("/callback", h.Callback)         // 微信回调
}

// WeChatPaymentHandler 处理微信支付相关的请求
type WeChatPaymentHandler struct {
	handler   *notify.Handler
	nativeSvc *service.NativePaymentService
}

func NewWeChatPaymentHandler(handler *notify.Handler, nativeSvc *service.NativePaymentService) *WeChatPaymentHandler {
	return &WeChatPaymentHandler{
		handler:   handler,
		nativeSvc: nativeSvc,
	}
}

// HandleNative 处理微信支付回调
func (h *WeChatPaymentHandler) HandleNative(ctx *gin.Context) {
	txn := new(payments.Transaction)
	// 从HTTP请求中解析出微信支付的回调数据
	_, err := h.handler.ParseNotifyRequest(ctx.Request.Context(), ctx.Request, txn)
	if err != nil {
		// 可能是因为有人伪造了请求
		ctx.JSON(400, gin.H{"error": "解析请求失败"})
		return
	}
	// 处理回调数据
	err = h.nativeSvc.HandleCallback(ctx, txn)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	// 返回成功响应
	// ctx.JSON(http.StatusOK, gin.H{
	// 	"message": "success",
	// })
	ctx.XML(http.StatusOK, map[string]string{
		"return_code": "SUCCESS",
		"return_msg":  "OK",
	})
}

// func (h *WechatHandler) AuthURL(c *gin.Context) {
// 	// 获取微信授权链接
// 	url, err := h.svc.AuthURL(c.Request.Context()) // 使用Request.Context()而不是直接传入c
// 	if err != nil {
// 		c.JSON(500, gin.H{"error": "获取微信授权链接失败"})
// 		return
// 	}
// 	c.JSON(200, gin.H{"url": url})
// }

// func (h *WechatHandler) Callback(c *gin.Context) {
// }