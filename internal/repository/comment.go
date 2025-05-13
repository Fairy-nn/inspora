package repository

import (
	"context"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/repository/cache"
	"github.com/Fairy-nn/inspora/internal/repository/dao"
)

type CommentRepository interface {
	// CreateComment 创建评论
	CreateComment(ctx context.Context, comment domain.Comment) (int64, error)
	// GetComment 根据ID获取评论
	GetComment(ctx context.Context, id int64) (domain.Comment, error)
	// GetRootComments 获取根评论列表
	GetRootComments(ctx context.Context, biz string, bizID int64, minID int64, limit int) ([]domain.Comment, error)
	// GetChildrenComments 获取子评论列表
	GetChildrenComments(ctx context.Context, parentID int64, minID int64, limit int) ([]domain.Comment, error)
	// DeleteComment 删除评论及其所有子评论
	DeleteComment(ctx context.Context, id int64) error
	// GetHotComments 获取热门评论（按子评论数量降序）
	GetHotComments(ctx context.Context, biz string, bizID int64, limit int) ([]domain.Comment, error)
	// PreloadArticleComments 预加载文章前三条评论及其子评论
	PreloadArticleComments(ctx context.Context, articleID int64) error
	// GetUserById 获取用户信息
	GetUserById(ctx context.Context, userID int64) (domain.User, error)
}

type CachedCommentRepository struct {
	dao   dao.CommentDAO
	cache cache.CommentCache
}

func NewCachedCommentRepository(dao dao.CommentDAO, cache cache.CommentCache) CommentRepository {
	return &CachedCommentRepository{
		dao:   dao,
		cache: cache,
	}
}

// CreateComment 创建评论
func (r *CachedCommentRepository) CreateComment(ctx context.Context, comment domain.Comment) (int64, error) {
	// 转换为DAO模型
	daoComment := dao.Comment{
		// ID:        comment.ID,
		ParentID: comment.ParentID,
		RootID:   comment.RootID,
		Biz:      comment.Biz,
		BizID:    comment.BizID,
		UserID:   comment.UserID,
		Content:  comment.Content,
		UserName: comment.UserName,
		Status:   uint8(domain.CommentStatusNormal),
		Ctime:    comment.Ctime,
	}

	// 插入数据库
	id, err := r.dao.Insert(ctx, daoComment)
	if err != nil {
		return 0, err
	}
	// 设置评论ID
	daoComment.ID = id
	// 更新缓存,新发布的评论暂存十五分钟
	err = r.cache.SetComment(ctx, daoComment, time.Minute*15)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetComment 根据ID获取评论
func (r *CachedCommentRepository) GetComment(ctx context.Context, id int64) (domain.Comment, error) {
	//先查缓存
	comment, err := r.cache.GetComment(ctx, id)
	if err != nil {
		return r.convertToModel(comment), nil
	}

	// 如果缓存不存在，则查询数据库
	comment, err = r.dao.GetByID(ctx, id)
	if err != nil {
		return domain.Comment{}, err
	}

	// 更新缓存
	err = r.cache.SetComment(ctx, comment, time.Minute*15)
	return r.convertToModel(comment), err
}

// GetRootComments 获取根评论列表
func (r *CachedCommentRepository) GetRootComments(ctx context.Context, biz string, bizID int64, minID int64, limit int) ([]domain.Comment, error) {
	// 查询数据库
	commentsDAO, err := r.dao.GetRootComments(ctx, biz, bizID, minID, limit)
	if err != nil {
		return nil, err
	}

	// 转换为领域模型
	comments := make([]domain.Comment, 0, len(commentsDAO))
	for _, commentDAO := range commentsDAO {
		comments = append(comments, r.convertToModel(commentDAO))

		// 同时更新缓存
		_ = r.cache.SetComment(ctx, commentDAO, time.Minute*15)
	}

	return comments, nil
}

// GetChildrenComments 获取子评论列表
func (r *CachedCommentRepository) GetChildrenComments(ctx context.Context, parentID int64, minID int64, limit int) ([]domain.Comment, error) {
	// 查询数据库
	commentsDAO, err := r.dao.GetChildrenComments(ctx, parentID, minID, limit)
	if err != nil {
		return nil, err
	}
	// 转换为领域模型
	comments := make([]domain.Comment, 0, len(commentsDAO))
	for _, commentDAO := range commentsDAO {
		comments = append(comments, r.convertToModel(commentDAO))

		// 同时更新缓存
		_ = r.cache.SetComment(ctx, commentDAO, time.Minute*15)
	}
	return comments, nil
}

// DeleteComment 删除评论及其所有子评论
func (r *CachedCommentRepository) DeleteComment(ctx context.Context, id int64) error {
	// 先获取评论，确认存在并获取相关信息
	comment, err := r.dao.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 删除数据库中的评论及其子评论
	if err := r.dao.DeleteByID(ctx, id); err != nil {
		return err
	}

	// 删除缓存
	_ = r.cache.DelComment(ctx, id)

	// 如果是文章评论，清除热门评论缓存
	if comment.Biz == "article" {
		_ = r.cache.DelHotComments(ctx, comment.Biz, comment.BizID)
	}

	return nil
}

// GetHotComments 获取热门评论（按子评论数量降序）
func (r *CachedCommentRepository) GetHotComments(ctx context.Context, biz string, bizID int64, limit int) ([]domain.Comment, error) {
	// 先查缓存
	commentDAO, err := r.cache.GetHotComments(ctx, biz, bizID)
	if err == nil && len(commentDAO) > 0 {
		// 转换为领域模型
		comments := make([]domain.Comment, 0, len(commentDAO))
		for _, comment := range commentDAO {
			comments = append(comments, r.convertToModel(comment))
		}
		return comments, nil
	}
	// 如果缓存不存在，则查询数据库
	commentsDAO, err := r.dao.GetHotComments(ctx, biz, bizID, limit)
	if err != nil {
		return nil, err
	}
	// 更新缓存
	_ = r.cache.SetHotComments(ctx, biz, bizID, commentsDAO, time.Minute*15)
	// 转换为领域模型
	comments := make([]domain.Comment, 0, len(commentsDAO))
	for _, commentDAO := range commentsDAO {
		comments = append(comments, r.convertToModel(commentDAO))
	}

	return comments, nil
}

// PreloadArticleComments 预加载文章前三条评论及其子评论
func (r *CachedCommentRepository) PreloadArticleComments(ctx context.Context, articleID int64) error {
	// 获取文章前三条热门评论
	// 获取文章前三条热门评论
	comments, err := r.dao.GetHotComments(ctx, "article", articleID, 3)
	if err != nil {
		return err
	}

	// 预加载到缓存
	return r.cache.PreloadComments(ctx, "article", articleID, comments)
}

func (r *CachedCommentRepository) convertToModel(comment dao.Comment) domain.Comment {
	return domain.Comment{
		ID:       comment.ID,
		Content:  comment.Content,
		ParentID: comment.ParentID,
		RootID:   comment.RootID,
		Biz:      comment.Biz,
		BizID:    comment.BizID,
		UserID:   comment.UserID,
		UserName: comment.UserName,
		//Status:    domain.CommentStatus(comment.Status),
		Ctime: comment.Ctime,
	}
}

// GetUserById 获取用户信息
func (r *CachedCommentRepository) GetUserById(ctx context.Context, userID int64) (domain.User, error){
	user, err := r.dao.GetUserById(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}
	return domain.User{
		ID:       user.ID,
		Username: user.Username,
	}, nil
}
