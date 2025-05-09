package web

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/service"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

// ArticleHandler 文章处理器
type ArticleHandler struct {
	svc            service.ArticleServiceInterface     // 文章服务
	interactionSvc service.InteractionServiceInterface // 交互服务
	biz            string                              // 业务线
	rankSvc        service.RankingServiceInterface     // 排行榜服务
}

// NewArticleHandler 创建文章处理器
func NewArticleHandler(svc service.ArticleServiceInterface,
	interactionSvc service.InteractionServiceInterface,
	rankSvc service.RankingServiceInterface) *ArticleHandler {
	return &ArticleHandler{
		svc:            svc,
		interactionSvc: interactionSvc,
		biz:            "article",
		rankSvc:        rankSvc,
	}
}

// RegisterRoutes 注册路由
func (a *ArticleHandler) RegisterRoutes(r *gin.Engine) {
	ag := r.Group("/article")        // 文章相关路由
	ag.POST("/edit", a.Edit)         // 创建文章
	ag.POST("/publish", a.Publish)   // 发布文章
	ag.POST("/withdraw", a.Withdraw) // 撤回文章
	ag.POST("/list", a.List)         // 文章列表
	ag.GET("/detail/:id", a.Detail)  // 文章详情,用户查看自己所有状态的文章

	pub := r.Group("/pub")          // 公开文章相关路由
	pub.GET("/:id", a.PubDetail)    // 发布文章详情，用户查看所有已公布的文章
	pub.POST("/like", a.Like)       // 点赞文章
	pub.POST("/collect", a.Collect) // 收藏文章
	pub.GET("/rank", a.Ranking)     // 文章排行榜

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
	ViewCount  int64  `json:"view_count"`  // 浏览量
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
	// 检查 userID 是否为 int64 类型
	userIDInt64, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Invalid user ID type"})
		return
	}
	// 调用服务层保存文章
	articleID, err := a.svc.Publish(c, domain.Article{
		ID:      req.ID,
		Title:   req.Title,
		Content: req.Content,
		Author: domain.Author{
			ID: userIDInt64, //作者ID
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
	articles, err := a.svc.List(c, userID.(int64), req.Limit, req.Offset)
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
		"message":  "success",
		"articles": articleVOs,
		"total":    len(articleVOs),
		//	"page":      req.Offset / req.Limit,
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

// Detail 文章详情
func (a *ArticleHandler) Detail(c *gin.Context) {
	// 获取文章ID
	idstr := c.Param("id")

	id, err := strconv.ParseInt(idstr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文章ID不合法"})
		return
	}
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "需要传入文章ID"})
		return
	}

	userID, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	// 调用服务层获取文章详情
	article, err := a.svc.FindById(c, id, userID.(int64))
	if err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "文章不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取文章失败"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "success",
		"article_id": id,
		"article":    toArticleVO(article),
	})
}

// PubDetail 发布文章详情
func (a *ArticleHandler) PubDetail(c *gin.Context) {
	// 获取文章ID
	idstr := c.Param("id")
	id, err := strconv.ParseInt(idstr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "文章ID不合法"})
		return
	}
	if id == 0 {
		c.JSON(400, gin.H{"error": "需要传入文章ID"})
		return
	}

	// 获取用户ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	// 创建一个 errgroup 用于并发执行多个任务并处理错误
	var eg errgroup.Group
	// 用于存储文章信息
	var art domain.Article
	// 存储交互信息
	var interaction domain.Interaction

	// 并发获取文章信息
	eg.Go(func() error {
		var err error
		art, err = a.svc.FindPublicArticleById(c, id, userID.(int64))
		return err
	})

	// 并发获取交互信息
	eg.Go(func() error {
		// 根据文章 ID 和用户 ID 获取交互信息
		interaction, err = a.interactionSvc.Get(c, a.biz, id, userID.(int64))
		fmt.Println("interaction:", interaction)
		if err != gorm.ErrRecordNotFound {
			return err
		}
		return nil
	})

	// 等待所有 goroutine 完成
	err = eg.Wait()
	// 错误处理部分
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "文章不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取文章失败"})
		}
		return
	}

	// go func() {
	// 	err := a.interactionSvc.IncrViewCount(c, a.biz, art.ID)
	// 	if err != nil {
	// 		fmt.Printf("增加文章浏览量失败:%v, id:%d\n", err, art.ID)
	// 	}
	// }()

	c.JSON(200, gin.H{
		"message":    "success",
		"article_id": art.ID,
		"article":    toArticleVO(art),
		"interaction": gin.H{
			"like_count":    interaction.LikeCnt,
			"collect_count": interaction.CollectCnt,
			"view_count":    interaction.ViewCnt,
			"liked":         interaction.Liked,
			"collected":     interaction.Collected,
		},
	})
}

// Like 点赞文章
func (a *ArticleHandler) Like(c *gin.Context) {
	type LikeRequest struct {
		ID   int64 `json:"id"`
		Like bool  `json:"like"`
	}
	var req LikeRequest

	if err := c.Bind(&req); err != nil {
		c.JSON(400, gin.H{"error": "请求无效"})
		return
	}

	if req.ID == 0 {
		c.JSON(400, gin.H{"error": "文章ID不能为空"})
		return
	}

	// 获取用户ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})

		fmt.Println("用户ID不存在")
		return
	}

	// 加了一个判断，如果用户已经点赞了，就不再执行点赞操作
	interaction, err := a.interactionSvc.Get(c, a.biz, req.ID, userID.(int64))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "交互信息不存在"})
			return
		}
	}
	if req.Like && interaction.Liked {
		c.JSON(200, gin.H{
			"message": "success",
			"liked":   true,
		})
		return
	} else if !req.Like && !interaction.Liked {
		c.JSON(200, gin.H{
			"message": "Already not liked",
			"liked":   false,
		})
		return
	}

	if req.Like { // true
		err = a.interactionSvc.Like(c, a.biz, req.ID, userID.(int64))
	} else { // false
		err = a.interactionSvc.CancelLike(c, a.biz, req.ID, userID.(int64))
	}

	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to like/dislike article"})
		return
	}

	c.JSON(200, gin.H{
		"message": "Success",
		"liked":   req.Like,
	})
}

// Collect 收藏文章
func (a *ArticleHandler) Collect(c *gin.Context) {
	type CollectRequest struct {
		ID      int64 `json:"id"`
		Collect bool  `json:"collect"`
	}

	var req CollectRequest

	if err := c.Bind(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	if req.ID == 0 {
		c.JSON(400, gin.H{"error": "Article ID is required"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var err error
	if req.Collect {
		err = a.interactionSvc.Collect(c, a.biz, req.ID, 0, userID.(int64))
	} else {
		err = a.interactionSvc.CancelCollect(c, a.biz, req.ID, 0, userID.(int64))
	}

	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to collect/uncollect article"})
		return
	}

	c.JSON(200, gin.H{
		"message": "Success",
	})
}

// Ranking 文章排行榜
func (a *ArticleHandler) Ranking(c *gin.Context) {
	// 调用服务层获取文章排行榜
	articles, err := a.rankSvc.GetTopN(c)
	if err != nil {
		fmt.Println("获取排行榜失败", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取排行榜失败"})
		return
	}

	// 如果获取到的文章为空，则主动触发排行榜计算
	if len(articles) == 0 {
		fmt.Println("排行榜为空，正在主动触发计算...")
		err := a.rankSvc.TopN(c)
		if err != nil {
			fmt.Println("计算排行榜失败", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "计算排行榜失败"})
			return
		}

		// 重新获取计算后的排行榜
		articles, err = a.rankSvc.GetTopN(c)
		if err != nil {
			fmt.Println("重新获取排行榜失败", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取排行榜失败"})
			return
		}
	}

	// 将文章列表转换为前端需要的格式
	articleVOs := make([]ArticleV0, 0, len(articles))
	for _, article := range articles {
		articleVOs = append(articleVOs, toArticleVO(article))
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "success",
		"articles": articleVOs,
		"total":    len(articleVOs),
	})
}
