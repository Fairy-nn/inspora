package web

import (
	"net/http"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/service"
	"github.com/gin-gonic/gin"
)

// ArticleHandler 文章处理器
type ArticleHandler struct {
	svc service.ArticleServiceInterface // 文章服务
}

// NewArticleHandler 创建文章处理器
func NewArticleHandler(svc service.ArticleServiceInterface) *ArticleHandler {
	return &ArticleHandler{
		svc: svc,
	}
}

// RegisterRoutes 注册路由
func (a *ArticleHandler) RegisterRoutes(r *gin.Engine) {
	ag := r.Group("/article") // 文章相关路由
	ag.POST("/edit", a.Edit)  // 创建文章
}

func (a *ArticleHandler) Edit(c *gin.Context) {
	type Request struct {
		ID      int64  `json:"id"`
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	var req Request
	if err := c.Bind(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	// 文章标题和内容不能为空
	if req.Title == "" || req.Content == "" {
		c.JSON(400, gin.H{"error": "title and content are required"})
		return
	}
	// 获取用户ID
	userID, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	// 断言为 float64
	userIDFloat, ok := userID.(float64)
	if !ok {
		// 处理类型不正确的情况，可能需要返回错误
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "用户 ID 类型错误"})
		return
	}

	// 调用服务层保存文章
	articleID, err := a.svc.Save(c, domain.Article{
		Title:   req.Title,
		Content: req.Content,
		Author: domain.Author{
			ID: int64(userIDFloat), 
		},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save article"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":    "success",
		"article_id": articleID,
	})
}
