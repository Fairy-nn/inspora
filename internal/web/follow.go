package web

import (
	"net/http"
	"strconv"

	"github.com/Fairy-nn/inspora/internal/service"
	"github.com/gin-gonic/gin"
)

type FollowHandler struct {
	svc service.FollowService
}

func NewFollowHandler(svc service.FollowService) *FollowHandler {
	return &FollowHandler{
		svc: svc,
	}
}

func (h *FollowHandler) RegisterRoutes(server *gin.Engine) {
	g := server.Group("/follow")
	g.POST("", h.Follow)                        // 关注别人
	g.DELETE("", h.CancelFollow)                // 取消关注
	g.GET("/relation", h.GetFollowRelation)     // 获取关注关系
	g.GET("/followees", h.GetFolloweeList)      // 获取关注列表
	g.GET("/followers", h.GetFollowerList)      // 获取粉丝列表
	g.GET("/statistics", h.GetFollowStatistics) // 获取关注统计信息
}

// 关注用户
func (h *FollowHandler) Follow(c *gin.Context) {
	type FollowRequest struct {
		Followee int64 `json:"followee" `
	}
	var req FollowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	// 获取当前用户的ID
	userID, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 关注用户
	if err := h.svc.Follow(c, userID.(int64), req.Followee); err != nil {
		c.JSON(500, gin.H{"error": "Failed to follow user"})
		return
	}

	c.JSON(200, gin.H{"message": "Followed user successfully"})
}

// 取消关注用户
func (h *FollowHandler) CancelFollow(c *gin.Context) {
	followeeStr := c.Query("followee")
	followee, err := strconv.ParseInt(followeeStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid followee ID"})
		return
	}

	// 获取当前用户的ID
	userID, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 取消关注用户
	if err := h.svc.CancelFollow(c, userID.(int64), followee); err != nil {
		c.JSON(500, gin.H{"error": "Failed to unfollow user"})
		return
	}

	c.JSON(200, gin.H{
		"message":  "Unfollowed user successfully",
		"followee": followee,
	})

}

// 获取关注关系,返回的是当前用户是否关注了指定用户
func (h *FollowHandler) GetFollowRelation(c *gin.Context) {
	followeeStr := c.Query("followee")
	followee, err := strconv.ParseInt(followeeStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Result{
			Code: 4,
			Msg:  "Invalid request parameters",
		})
		return
	}

	// 获取当前用户的ID
	uid := c.GetInt64("userID")
	if uid <= 0 {
		c.JSON(http.StatusUnauthorized, Result{
			Code: 5,
			Msg:  "Not logged in",
		})
		return
	}

	// 获取关注关系
	isFollowing, err := h.svc.GetFollowInfo(c, uid, followee)
	if err != nil {
		c.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Result{
		Data: gin.H{
			"is_following": isFollowing,
		},
	})

}

// GetFolloweeList 获取某个用户的关注列表
func (h *FollowHandler) GetFolloweeList(ctx *gin.Context) {
	// 解析分页参数
	offset, limit := extractPaginationParams(ctx)

	// 获取用户ID，可以是当前用户也可以是查询的目标用户
	userIdStr := ctx.Query("user_id")
	var userId int64
	var err error
	if userIdStr == "" {
		userId = ctx.GetInt64("userID")
		if userId <= 0 {
			ctx.JSON(http.StatusUnauthorized, Result{
				Code: 5,
				Msg:  "Not logged in",
			})
			return
		}
	} else {
		userId, err = strconv.ParseInt(userIdStr, 10, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, Result{
				Code: 4,
				Msg:  "Invalid request parameters",
			})
			return
		}
	}

	// 获取关注列表
	followees, err := h.svc.GetFolloweeList(ctx, userId, offset, limit)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  err.Error(),
		})
		return
	}

	// 返回结果
	ctx.JSON(http.StatusOK, Result{
		Data: followees,
	})
}

// GetFollowerList 获取某个用户的粉丝列表
func (h *FollowHandler) GetFollowerList(ctx *gin.Context) {
	// 解析分页参数
	offset, limit := extractPaginationParams(ctx)

	// 获取用户ID，可以是当前用户也可以是查询的目标用户
	userIdStr := ctx.Query("user_id")
	var userId int64
	var err error
	if userIdStr == "" {
		userId = ctx.GetInt64("userID")
		if userId <= 0 {
			ctx.JSON(http.StatusUnauthorized, Result{
				Code: 5,
				Msg:  "Not logged in",
			})
			return
		}
	} else {
		userId, err = strconv.ParseInt(userIdStr, 10, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, Result{
				Code: 4,
				Msg:  "Invalid request parameters",
			})
			return
		}
	}

	// 获取粉丝列表
	followers, err := h.svc.GetFollowerList(ctx, userId, offset, limit)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  err.Error(),
		})
		return
	}

	// 返回结果
	ctx.JSON(http.StatusOK, Result{
		Data: followers,
	})
}

// GetFollowStatistics 获取关注统计信息
// 返回的是当前用户的关注和粉丝数量
func (h *FollowHandler) GetFollowStatistics(ctx *gin.Context) {
	// 获取用户ID，可以是当前用户也可以是查询的目标用户
	userIdStr := ctx.Query("user_id")
	var userId int64
	var err error
	if userIdStr == "" {
		userId = ctx.GetInt64("userID")
		if userId <= 0 {
			ctx.JSON(http.StatusUnauthorized, Result{
				Code: 5,
				Msg:  "Not logged in",
			})
			return
		}
	} else {
		userId, err = strconv.ParseInt(userIdStr, 10, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, Result{
				Code: 4,
				Msg:  "Invalid request parameters",
			})
			return
		}
	}

	// 获取统计数据
	statistics, err := h.svc.GetFollowStatistics(ctx, userId)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  err.Error(),
		})
		return
	}

	// 返回结果
	ctx.JSON(http.StatusOK, Result{
		Data: statistics,
	})
}

func extractPaginationParams(ctx *gin.Context) (int64, int64) {
	pageStr := ctx.Query("page")
	pageSizeStr := ctx.Query("page_size")

	var page int64 = 1
	var pageSize int64 = 10

	if pageStr != "" {
		parsedPage, err := strconv.ParseInt(pageStr, 10, 64)
		if err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	if pageSizeStr != "" {
		parsedPageSize, err := strconv.ParseInt(pageSizeStr, 10, 64)
		if err == nil && parsedPageSize > 0 {
			pageSize = parsedPageSize
			if pageSize > 100 {
				pageSize = 100
			}
		}
	}

	offset := (page - 1) * pageSize
	return offset, pageSize
}
