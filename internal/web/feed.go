package web

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Fairy-nn/inspora/internal/service"
	"github.com/gin-gonic/gin"
)

type FeedHandler struct {
	svc service.FeedServiceInterface
}

func NewFeedHandler(svc service.FeedServiceInterface) *FeedHandler {
	return &FeedHandler{
		svc: svc,
	}
}

func (h *FeedHandler) RegisterRoutes(server *gin.Engine) {
	g := server.Group("/feeds")
	// 用户获取当前用户的 feeds
	g.GET("", h.GetUserFeed)
	// 被动触发重建功能
	g.POST("/rebuild", h.RebuildFeed)
}

// GetUserFeed 处理获取用户feed流的http请求
func (h *FeedHandler) GetUserFeed(c *gin.Context) {
	// 获取当前用户的ID
	userID, ok := c.Get("userID")
	if !ok {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}

	// 获取分页参数
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil {
		// 解析失败时使用默认值
		offset = 0
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil {
		limit = 20
	}
	// 限制最大每页条数，防止恶意请求
	if limit > 100 {
		limit = 100
	}

	// 调用服务层获取用户的feed流
	feeds, err := h.svc.GetUserFeed(c, userID.(int64), offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Result{
			Code: 500,
			Msg:  "获取 Feed 失败: " + err.Error(),
		})
		return
	}

	// 成功响应，返回数据
	c.JSON(http.StatusOK, Result{
		Code: 200,
		Msg:  "获取 Feed 成功",
		Data: feeds,
	})
}

// RebuildFeed 重建用户的Feed
func (h *FeedHandler) RebuildFeed(ctx *gin.Context) {
	// 从token中获取用户ID
	userID, ok := ctx.Get("userID")
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "未授权",
		})
		return
	}

	// 获取重建的天数范围
	var req struct {
		SinceDays int `json:"since_days"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "参数错误",
		})
		return
	}

	// 默认重建最近30天的Feed
	if req.SinceDays <= 0 {
		req.SinceDays = 30
	}

	// 限制最大天数为90天
	if req.SinceDays > 90 {
		req.SinceDays = 90
	}

	// 在后台异步执行重建任务
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if err := h.svc.RebuildUserFeed(ctx, userID.(int64), req.SinceDays); err != nil {
			fmt.Printf("重建用户 %d 的Feed失败: %v\n", userID, err)
		}
	}()

	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "Feed重建任务已启动，请稍后刷新查看结果",
	})
}
