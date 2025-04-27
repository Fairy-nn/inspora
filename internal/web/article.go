package web

import (
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

}
