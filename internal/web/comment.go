package web

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/service"
	"github.com/gin-gonic/gin"
)

// Result 是统一响应结构
type Result struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

type CommentHandler struct {
	svc service.CommentService
}

func NewCommentHandler(svc service.CommentService) *CommentHandler {
	return &CommentHandler{
		svc: svc,
	}
}

// RegisterRoutes 注册路由
func (h *CommentHandler) RegisterRoutes(server *gin.Engine) {
	g := server.Group("/comments")
	g.POST("", h.CreateComment)
	g.DELETE("/:id", h.DeleteComment)
	// GET /articles/456?min_id=10&limit=15 HTTP/1.1
	g.GET("/articles/:articleId", h.GetArticleComments)
	g.GET("/:id/children", h.GetChildrenComments)
	g.GET("/hot/articles/:articleId", h.GetHotComments)
}

// 创建评论的请求参数
type CreateCommentReq struct {
	Content  string `json:"content" binding:"required"`
	ParentID int64  `json:"parent_id"` // 可选，父评论ID，0表示根评论
	Biz      string `json:"biz" binding:"required"`
	BizID    int64  `json:"biz_id" binding:"required"`
}

// CreateComment 创建评论
func (h *CommentHandler) CreateComment(ctx *gin.Context) {
	// 获取用户ID
	userID, ok := ctx.Get("userID")
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req CreateCommentReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, Result{
			Code: 400,
			Msg:  "invalid request"})
		return
	}
	// 调用服务层获取用户Name
	userName, _ := h.svc.GetUserNameById(ctx, userID.(int64))

	comment := domain.Comment{
		Content:  req.Content,
		UserID:   userID.(int64),
		UserName: userName,
		Biz:      req.Biz,
		BizID:    req.BizID,
		ParentID: req.ParentID,
		Ctime:    time.Now().Unix(),
	}
	// 调用服务层创建评论
	id, err := h.svc.CreateComment(ctx, comment)
	if err != nil {
		if errors.Is(err, service.ErrInvalidComment) {
			ctx.JSON(http.StatusBadRequest, Result{
				Code: 400,
				Msg:  "评论内容不能为空",
			})
			return
		}
		if errors.Is(err, service.ErrCommentNotFound) {
			ctx.JSON(http.StatusBadRequest, Result{
				Code: 400,
				Msg:  "父评论不存在",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 500,
			Msg:  "系统错误",
		})
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Data: id,
	})
}

// DeleteComment 删除评论
func (h *CommentHandler) DeleteComment(ctx *gin.Context) {
	// 获取用户ID
	userID, ok := ctx.Get("userID")
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	// 获取评论ID
	commentID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, Result{
			Code: 400,
			Msg:  "无效的评论ID",
		})
		return
	}
	// 调用服务层删除评论
	err = h.svc.DeleteComment(ctx, commentID, userID.(int64))
	if err != nil {
		if errors.Is(err, service.ErrCommentNotFound) {
			ctx.JSON(http.StatusNotFound, Result{
				Code: 404,
				Msg:  "评论不存在",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 500,
			Msg:  "系统错误",
		})
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Msg: "删除成功",
	})
}

// CommentResp 评论响应
type CommentResp struct {
	ID       int64         `json:"id"`
	Content  string        `json:"content"`
	UserID   int64         `json:"user_id"`
	UserName string        `json:"user_name"`
	ParentID int64         `json:"parent_id,omitempty"`
	RootID   int64         `json:"root_id,omitempty"`
	Ctime    int64         `json:"ctime"`
	Children []CommentResp `json:"children,omitempty"`
}

// GetArticleComments 获取文章评论
func (h *CommentHandler) GetArticleComments(ctx *gin.Context) {
	// 获取文章ID
	articleIdStr := ctx.Param("articleId")
	articleId, err := strconv.ParseInt(articleIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, Result{
			Code: 400,
			Msg:  "invalid article id",
		})
		return
	}
	// 获取分页参数:min_id和limit
	// min_id: 从哪个ID开始获取评论，默认0
	// limit: 每页获取多少条评论，默认20
	minIDStr := ctx.Query("min_id")
	var minID int64 = 0
	if minIDStr != "" {
		minID, err = strconv.ParseInt(minIDStr, 10, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, Result{
				Code: 400,
				Msg:  "invalid min_id",
			})
			return
		}
	}
	limitStr := ctx.Query("limit")
	var limit int = 20
	if limitStr != "" {
		limit64, err := strconv.ParseInt(limitStr, 10, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, Result{
				Code: 400,
				Msg:  "invalid limit",
			})
			return
		}
		limit = int(limit64)
	}

	// 获取评论列表
	comments, err := h.svc.GetComments(ctx, "article", articleId, minID, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 500,
			Msg:  "系统错误",
		})
		return
	}

	// 转换为响应格式
	resp := make([]CommentResp, 0, len(comments))
	for _, comment := range comments {
		commentResp := CommentResp{
			ID:       comment.ID,
			Content:  comment.Content,
			UserID:   comment.UserID,
			UserName: comment.UserName,
			ParentID: comment.ParentID,
			RootID:   comment.RootID,
			Ctime:    comment.Ctime,
		}
		// 处理子评论
		if len(comment.Children) > 0 {
			children := make([]CommentResp, 0, len(comment.Children))
			for _, child := range comment.Children {
				children = append(children, CommentResp{
					ID:       child.ID,
					Content:  child.Content,
					UserID:   child.UserID,
					UserName: child.UserName,
					ParentID: child.ParentID,
					RootID:   child.RootID,
					Ctime:    child.Ctime,
				})
			}
			commentResp.Children = children
		}

		resp = append(resp, commentResp)
	}

	ctx.JSON(http.StatusOK, Result{
		Data: resp,
	})
}

// GetChildrenComments 获取子评论
func (h *CommentHandler) GetChildrenComments(ctx *gin.Context) {
	// 获取评论ID
	commentIDStr := ctx.Param("id")
	commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, Result{
			Code: 400,
			Msg:  "invalid comment id",
		})
		return
	}

	// 获取分页参数:min_id和limit
	minIDStr := ctx.Query("min_id")
	var minID int64 = 0
	if minIDStr != "" {
		minID, err = strconv.ParseInt(minIDStr, 10, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, Result{
				Code: 400,
				Msg:  "invalid min_id",
			})
			return
		}
	}
	limitStr := ctx.Query("limit")
	var limit int = 20
	if limitStr != "" {
		limit64, err := strconv.ParseInt(limitStr, 10, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, Result{
				Code: 400,
				Msg:  "invalid limit",
			})
			return
		}
		limit = int(limit64)
	}
	// 获取子评论列表
	childrenComments, err := h.svc.GetChildrenComments(ctx, commentID, minID, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 500,
			Msg:  "系统错误",
		})
		return
	}
	// 转换为响应格式
	resp := make([]CommentResp, len(childrenComments))
	for i, child := range childrenComments {
		resp[i] = CommentResp{
			ID:       child.ID,
			Content:  child.Content,
			UserID:   child.UserID,
			UserName: child.UserName,
			Ctime:    child.Ctime,
			ParentID: child.ParentID,
			RootID:   child.RootID,
		}
	}

	ctx.JSON(http.StatusOK, Result{
		Data: resp,
	})
}

// GetHotComments 获取热门评论
func (h *CommentHandler) GetHotComments(ctx *gin.Context) {
	// 获取文章ID
	articleIdStr := ctx.Param("articleId")
	articleId, err := strconv.ParseInt(articleIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, Result{
			Code: 400,
			Msg:  "invalid article id",
		})
		return
	}
	// 获取热门评论列表
	hotComments, err := h.svc.GetHotComments(ctx, "article", articleId, 3)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 500,
			Msg:  "系统错误",
		})
		return
	}
	// 转换为响应格式
	resp := make([]CommentResp, len(hotComments))
	for i, hotComment := range hotComments {
		resp[i] = CommentResp{
			ID:       hotComment.ID,
			Content:  hotComment.Content,
			UserID:   hotComment.UserID,
			UserName: hotComment.UserName,
			Ctime:    hotComment.Ctime,
			ParentID: hotComment.ParentID,
			RootID:   hotComment.RootID,
		}
		// 处理子评论
		if len(hotComment.Children) > 0 {
			children := make([]CommentResp, 0, len(hotComment.Children))
			for _, child := range hotComment.Children {
				children = append(children, CommentResp{
					ID:       child.ID,
					Content:  child.Content,
					UserID:   child.UserID,
					UserName: child.UserName,
					ParentID: child.ParentID,
					RootID:   child.RootID,
					Ctime:    child.Ctime,
				})
			}
			resp[i].Children = children
		}
	}

	ctx.JSON(http.StatusOK, Result{
		Data: resp,
	})
}
