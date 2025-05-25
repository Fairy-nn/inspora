package service

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

type OSSServiceInterface interface {
	// UploadFile 上传单个文件
	UploadFile(ctx context.Context, file *multipart.FileHeader) (string, error)
	// UploadFiles 上传多个文件
	UploadFiles(ctx context.Context, files []*multipart.FileHeader, maxFiles int) ([]string, error)
	// DeleteFile 根据文件URL删除文件
	DeleteFile(ctx context.Context, fileURL string) error
}

type OSSService struct {
	client     *oss.Client // 阿里云OSS客户端
	bucket     *oss.Bucket // 操作的目标存储空间
	bucketName string      // 存储空间名称
	baseURL    string      // 文件访问的基础URL
}

// NewOSSService 服务初始化
func NewOSSService() (OSSServiceInterface, error) {
	// 从配置文件中获取OSS相关配置
	endpoint := viper.GetString("oss.endpoint")
	accessKeyID := viper.GetString("oss.access_key_id")
	accessKeySecret := viper.GetString("oss.access_key_secret")
	bucketName := viper.GetString("oss.bucket_name")
	baseURL := viper.GetString("oss.base_url")

	// 创建OSS客户端
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create OSS client: %w", err)
	}

	// 获取存储空间
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	return &OSSService{
		client:     client,
		bucket:     bucket,
		bucketName: bucketName,
		baseURL:    baseURL,
	}, nil
}

// UploadFile 上传单个文件
func (s *OSSService) UploadFile(ctx context.Context, file *multipart.FileHeader) (string, error) {
	// 1. 打开上传的文件
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// 2. 校验文件类型（仅允许图片）
	fileExt := filepath.Ext(file.Filename)
	// 通过isValidImageExtension函数限制仅允许上传图片格式（jpg、png 等）
	if !isValidImageExtension(fileExt) {
		return "", errors.New("invalid file type")
	}

	// 3. 生成唯一文件名和存储路径（按日期组织）
	// 使用 UUID 生成唯一文件名，避免冲突
	// 按日期（年 / 月 / 日）组织文件，提高存储可读性
	filename := uuid.New().String() + fileExt
	objectKey := fmt.Sprintf("articles/%s/%s", time.Now().Format("2006/01/02"), filename)

	// 4. 上传文件到OSS
	err = s.bucket.PutObject(objectKey, src)

	// 5. 返回文件公共URL
	return fmt.Sprintf("%s/%s", s.baseURL, objectKey), nil
}

// UploadFiles 上传多个文件
func (s *OSSService) UploadFiles(ctx context.Context, files []*multipart.FileHeader, maxFiles int) ([]string, error) {
	// 校验文件数量是否超过限制
	if len(files) > maxFiles {
		return nil, fmt.Errorf("too many files")
	}
	// 存储上传的文件URL
	urls := make([]string, 0, len(files))
	for _, file := range files {
		// 上传每个文件
		url, err := s.UploadFile(ctx, file)
		if err != nil {
			return nil, fmt.Errorf("failed to upload file: %w", err)
		}
		urls = append(urls, url)
	}
	// 返回所有文件的公共URL
	return urls, nil
}

// DeleteFile 根据文件URL删除文件
func (s *OSSService) DeleteFile(ctx context.Context, fileURL string) error {
	// 从URL中提取ObjectKey
	objectKey, err := s.getObjectKeyFromURL(fileURL)

	// 删除文件
	err = s.bucket.DeleteObject(objectKey)

	return err
}

// isValidImageExtension 校验文件扩展名是否为图片格式
func isValidImageExtension(ext string) bool {
	// 允许的图片扩展名
	allowedExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp"}
	for _, allowedExt := range allowedExtensions {
		if ext == allowedExt {
			return true
		}
	}
	return false

}

// getObjectKeyFromURL 从文件URL中提取ObjectKey
// 只处理属于当前 Bucket 的 URL，防止越权删除
func (s *OSSService) getObjectKeyFromURL(fileURL string) (string, error) {
	if !strings.HasPrefix(fileURL, s.baseURL) {
		return "", errors.New("URL does not belong to this OSS bucket")
	}

	objectKey := strings.TrimPrefix(fileURL, s.baseURL+"/")
	if objectKey == "" {
		return "", errors.New("invalid file URL")
	}

	return objectKey, nil

}
