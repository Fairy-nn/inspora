package service

import (
	"context"
	"errors"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/repository"
)

var (
	ErrInvalidComment  = errors.New("invalid comment")
	ErrCommentNotFound = errors.New("comment not found")
)

type CommentService interface {
	// CreateComment 创建评论
	CreateComment(ctx context.Context, comment domain.Comment) (int64, error)
	// GetComment 根据ID获取评论
	GetComment(ctx context.Context, id int64) (domain.Comment, error)
	// GetComments 获取评论列表，包括子评论
	GetComments(ctx context.Context, biz string, bizID int64, minID int64, limit int) ([]domain.Comment, error)
	// GetChildrenComments 获取子评论列表
	GetChildrenComments(ctx context.Context, parentID int64, minID int64, limit int) ([]domain.Comment, error)
	// DeleteComment 删除评论
	DeleteComment(ctx context.Context, id int64, userID int64) error
	// GetHotComments 获取热门评论
	GetHotComments(ctx context.Context, biz string, bizID int64, limit int) ([]domain.Comment, error)
	// GetUserNameById 获取用户名称
	GetUserNameById(ctx context.Context, userID int64) (string, error)
}

type commentService struct {
	repo repository.CommentRepository
}

func NewCommentService(repo repository.CommentRepository) CommentService {
	return &commentService{
		repo: repo,
	}
}

// CreateComment 创建评论
func (s *commentService) CreateComment(ctx context.Context, comment domain.Comment) (int64, error) {
	// 参数校验
	if comment.Biz == "" || comment.BizID <= 0 || comment.Content == "" {
		return 0, ErrInvalidComment
	}
	// 如果是子评论，需要校验父评论是否存在
	if comment.ParentID > 0 {
		parent, err := s.repo.GetComment(ctx, comment.ParentID)
		if err != nil {
			return 0, ErrCommentNotFound
		}

		// 设置根评论ID
		if parent.RootID > 0 {
			comment.RootID = parent.RootID
		} else {
			// 否则父评论就是根评论
			comment.RootID = parent.ID
		}
	}
	return s.repo.CreateComment(ctx, comment)
}

// GetComment 根据ID获取评论
func (s *commentService) GetComment(ctx context.Context, id int64) (domain.Comment, error) {
	return s.repo.GetComment(ctx, id)
}

// GetComments 获取评论列表，包括子评论
func (s *commentService) GetComments(ctx context.Context, biz string, bizID int64, minID int64, limit int) ([]domain.Comment, error) {
	// 默认限制为20条
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// 获取根评论列表
	comments, err := s.repo.GetRootComments(ctx, biz, bizID, minID, limit)
	if err != nil {
		return nil, err
	}

	// 为每个根评论加载前三条子评论
	for i := range comments {
		children, err := s.repo.GetChildrenComments(ctx, comments[i].ID, 0, 3)
		if err != nil {
			return nil, err
		}
		// 转换为指针类型
		ptrChildren := make([]*domain.Comment, 0, len(children))
		for j := range children {
			ptrChildren = append(ptrChildren, &children[j])
		}
		comments[i].Children = ptrChildren
	}
	return comments, nil
}

// GetChildrenComments 获取子评论列表
func (s *commentService) GetChildrenComments(ctx context.Context, parentID int64, minID int64, limit int) ([]domain.Comment, error) {
	// 默认限制为20条
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.GetChildrenComments(ctx, parentID, minID, limit)
}

// DeleteComment 删除评论
func (s *commentService) DeleteComment(ctx context.Context, id int64, userID int64) error {
	// 检查评论是否存在，并且是否属于当前用户
	comment, err := s.repo.GetComment(ctx, id)
	if err != nil {
		return ErrCommentNotFound
	}
	if comment.UserID != userID {
		return errors.New("没有权限删除该评论")
	}
	// 删除评论及其所有子评论
	return s.repo.DeleteComment(ctx, id)
}

// GetHotComments 获取热门评论
func (s *commentService) GetHotComments(ctx context.Context, biz string, bizID int64, limit int) ([]domain.Comment, error) {
	// 默认获取3条热门评论
	if limit <= 0 || limit > 100 {
		limit = 3
	}
	comments, err := s.repo.GetHotComments(ctx, biz, bizID, limit)
	if err != nil {
		return nil, err
	}
	// 为每个热门评论加载前三条子评论
	for i := range comments {
		children, err := s.repo.GetChildrenComments(ctx, comments[i].ID, 0, 3)
		if err != nil {
			return nil, err
		}
		// 转换为指针类型
		ptrChildren := make([]*domain.Comment, 0, len(children))
		for j := range children {
			ptrChildren = append(ptrChildren, &children[j])
		}
		comments[i].Children = ptrChildren
	}
	return comments, nil
}

// GetUserNameById 获取用户名称
func (s *commentService) GetUserNameById(ctx context.Context, userID int64) (string, error) {
	user, err := s.repo.GetUserById(ctx, userID)
	if err != nil {
		return "", err
	}
	return user.Name, nil
}