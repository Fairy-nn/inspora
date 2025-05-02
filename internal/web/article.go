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
	ag := r.Group("/article")        // 文章相关路由
	ag.POST("/edit", a.Edit)         // 创建文章
	ag.POST("/publish", a.Publish)   // 发布文章
	ag.POST("/withdraw", a.Withdraw) // 撤回文章
	ag.POST("/list", a.List)         // 文章列表
}

// 前端的请求体
type Request struct {
	ID      int64  `json:"id"` //文章ID
	Title   string `json:"title"`
	Content string `json:"content"`
}

// 文章列表请求体
type ListRequest struct {
	Limit  int `json:"limit"`  // 每页数量
	Offset int `json:"offset"` // 偏移量
}

type ArticleV0 struct {
	ID         int64  `json:"id"`          //文章ID
	Title      string `json:"title"`       // 文章标题
	Abstract   string `json:"abstract"`    // 文章摘要
	Content    string `json:"content"`     // 文章内容
	AuthorID   int64  `json:"author_id"`   // 作者ID
	AuthorName string `json:"author_name"` // 作者名称
	Status     uint8  `json:"status"`      // 文章状态
	Ctime      int64  `json:"ctime"`       // 创建时间
	Utime      int64  `json:"utime"`       // 更新时间
}

// Edit 编辑文章
func (a *ArticleHandler) Edit(c *gin.Context) {
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
	userIDfloat, _ := userID.(float64)

	// 调用服务层保存文章
	articleID, err := a.svc.Save(c, domain.Article{
		Title:   req.Title,
		Content: req.Content,
		Author: domain.Author{
			ID: int64(userIDfloat), //作者ID
		},
		ID: req.ID, // 文章ID
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

// Publish 发布文章
func (a *ArticleHandler) Publish(c *gin.Context) {
	// 请求体结构体
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
	userIDfloat, _ := userID.(float64)
	// 调用服务层保存文章
	articleID, err := a.svc.Publish(c, domain.Article{
		ID:      req.ID,
		Title:   req.Title,
		Content: req.Content,
		Author: domain.Author{
			ID: int64(userIDfloat), //作者ID
		},
	})
	// 保存失败
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save article"})
		return
	}
	// 保存成功
	c.JSON(http.StatusOK, gin.H{
		"message":    "success",
		"article_id": articleID,
	})
}

// Withdraw 撤回文章
func (a *ArticleHandler) Withdraw(c *gin.Context) {
	// 请求体结构体
	type Req struct {
		ID int64 `json:"id"` // 文章ID
	}
	var req Req
	if err := c.Bind(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	// 文章ID不能为空
	if req.ID == 0 {
		c.JSON(400, gin.H{"error": "id is required"})
		return
	}

	// 获取用户ID
	userID, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 调用服务层撤回文章方法
	err := a.svc.Withdraw(c, domain.Article{
		ID: req.ID,
		Author: domain.Author{
			ID: int64(userID.(float64)), // 作者ID
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to withdraw article"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":    "success",
		"action":     "withdraw",
		"article_id": req.ID,
	})
}

// List 文章列表
func (a *ArticleHandler) List(c *gin.Context) {
	var req ListRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	// 检查limit的值
	if req.Limit <= 0 {
		req.Limit = 10 // 默认值
	}

	// 获取用户ID
	userID, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 调用服务层获取文章列表
	articles, err := a.svc.List(c, int64(userID.(float64)), req.Limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get article list"})
		return
	}

	// 将文章列表转换为前端需要的格式
	articleVOs := make([]ArticleV0, 0, len(articles))
	for _, article := range articles {
		articleVOs = append(articleVOs, toArticleVO(article))
	}

	// 返回文章列表
	c.JSON(http.StatusOK, gin.H{
		"message":   "success",
		//"articles":  articleVOs,
		"total":     len(articleVOs),
		"page":      req.Offset / req.Limit,
		"page_size": req.Limit,
	})
}

// toArticleVO 将文章转换为前端需要的格式
func toArticleVO(article domain.Article) ArticleV0 {
	return ArticleV0{
		ID:         article.ID,
		Title:      article.Title,
		Content:    article.Content,
		AuthorID:   article.Author.ID,
		AuthorName: article.Author.Name,
		Status:     article.Status.ToUint8(),
		Abstract:   article.GenerateAbstract(),
		Ctime:      article.Ctime.UnixMilli(),
		Utime:      article.Utime.UnixMilli(),
	}
}
