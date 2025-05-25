package web

import (
	"net/http"
	"strconv"

	"github.com/Fairy-nn/inspora/internal/service"
	"github.com/Fairy-nn/inspora/pkg/elasticsearch"
	"github.com/gin-gonic/gin"
)

// SearchHandler 搜索处理器
type SearchHandler struct {
	svc service.SearchService
}

// NewSearchHandler 创建搜索处理器
func NewSearchHandler(svc service.SearchService) *SearchHandler {
	return &SearchHandler{
		svc: svc,
	}
}

// RegisterRoutes 注册路由
func (h *SearchHandler) RegisterRoutes(server *gin.Engine) {
	group := server.Group("/search")
	group.GET("/users", h.SearchUsers)
	group.GET("/articles", h.SearchArticles)
	group.GET("/articles/author/:id", h.SearchArticlesByAuthor)
}

// UserSearchVO 用户搜索结果VO
type UserSearchVO struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

func (h *SearchHandler) SearchUsers(ctx *gin.Context) {
	// 搜索关键词
	query := ctx.Query("query")
	if query == "" {
		ctx.JSON(http.StatusBadRequest, Result{
			Code: 400,
			Msg:  "Search query is required",
		})
		return
	}

	// 解析分页参数
	page, pageSize := h.parsePagination(ctx)
	// 调用服务层执行搜索逻辑
	users, total, err := h.svc.SearchUsers(ctx, query, page, pageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 500,
			Msg:  "Failed to search users: " + err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, Result{
		Code: 200,
		Data: gin.H{
			"total": total,
			"users": func() []UserSearchVO {
				result := make([]UserSearchVO, 0, len(users))
				for _, user := range users {
					vo := UserSearchVO{
						ID:       user.ID,
						Username: user.Username,
						Email:    user.Email,
						Phone:    user.Phone,
					}
					result = append(result, vo)
				}
				return result
			}(),
		},
	})
}

func (h *SearchHandler) SearchArticles(ctx *gin.Context) {
	query := ctx.Query("query")
	if query == "" {
		ctx.JSON(http.StatusBadRequest, Result{
			Code: 400,
			Msg:  "Search query is required",
		})
		return
	}

	// 解析分页参数
	page, pageSize := h.parsePagination(ctx)

	articles, total, err := h.svc.SearchArticles(ctx, query, page, pageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 500,
			Msg:  "Failed to search articles: " + err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, Result{
		Code: 200,
		Data: gin.H{
			"total":    total,
			"articles": h.convertArticleResults(articles),
		},
	})
}

// SearchArticlesByAuthor 按作者搜索文章
// @Summary Search articles by author
// @Description Search articles by query and author ID
// @Tags Search
// @Accept json
// @Produce json
// @Param id path int true "Author ID"
// @Param query query string true "Search query"
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Page size (default 10)"
// @Success 200 {object} Result
// @Router /search/articles/author/{id} [get]
func (h *SearchHandler) SearchArticlesByAuthor(ctx *gin.Context) {
	idStr := ctx.Param("id")
	authorID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, Result{
			Code: 400,
			Msg:  "Invalid author ID",
		})
		return
	}

	query := ctx.Query("query")
	if query == "" {
		ctx.JSON(http.StatusBadRequest, Result{
			Code: 400,
			Msg:  "Search query is required",
		})
		return
	}

	// 解析分页参数
	page, pageSize := h.parsePagination(ctx)

	articles, total, err := h.svc.SearchArticlesByAuthor(ctx, query, authorID, page, pageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 500,
			Msg:  "Failed to search articles: " + err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, Result{
		Code: 200,
		Data: gin.H{
			"total":    total,
			"articles": h.convertArticleResults(articles),
		},
	})
}

// parsePagination 解析分页参数
func (h *SearchHandler) parsePagination(ctx *gin.Context) (int, int) {
	pageStr := ctx.DefaultQuery("page", "1")
	pageSizeStr := ctx.DefaultQuery("page_size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	return page, pageSize
}

// ArticleSearchVO 文章搜索结果VO
type ArticleSearchVO struct {
	ID         int64               `json:"id"`
	Title      string              `json:"title"`
	Abstract   string              `json:"abstract"`
	Author     AuthorVO            `json:"author"`
	Status     string              `json:"status"`
	Ctime      int64               `json:"ctime"`
	Utime      int64               `json:"utime"`
	Highlights map[string][]string `json:"highlights"`
}

// AuthorVO 作者VO
type AuthorVO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// convertArticleResults 转换文章搜索结果
func (h *SearchHandler) convertArticleResults(articles []elasticsearch.ArticleSearchResult) []ArticleSearchVO {
	result := make([]ArticleSearchVO, 0, len(articles))

	for _, article := range articles {
		vo := ArticleSearchVO{
			ID:       article.ID,
			Title:    article.Title,
			Abstract: article.Abstract,
			Author: AuthorVO{
				ID:   article.Author.ID,
				Name: article.Author.Name,
			},
			Status:     article.Status.String(),
			Ctime:      article.Ctime.Unix(),
			Utime:      article.Utime.Unix(),
			Highlights: article.Highlights,
		}

		result = append(result, vo)
	}

	return result
}
