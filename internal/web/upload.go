package web

import (
	"fmt"
	"net/http"

	"github.com/Fairy-nn/inspora/internal/service"
	"github.com/gin-gonic/gin"
)

const maxArticleImages = 9 // 限制了一次最多上传 9 张图片

type UploadHandler struct {
	svc service.OSSServiceInterface
}

func NewUploadHandler(svc service.OSSServiceInterface) *UploadHandler {
	return &UploadHandler{
		svc: svc,
	}
}

func (h *UploadHandler) RegisterRoutes(server *gin.Engine) {
	g := server.Group("/upload")
	g.POST("/article/image", h.UploadArticleImage)
	g.POST("/article/images", h.UploadArticleImages)
}

// UploadArticleImages 单图上传处理
func (h *UploadHandler) UploadArticleImage(c *gin.Context) {
	// 1. 身份验证
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 2. 获取上传的文件
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image file uploaded"})
		return
	}

	// 3. 文件大小校验（10MB限制）
	if file.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large (max 10MB)"})
		return
	}

	// 4. 调用OSS服务上传文件
	url, err := h.svc.UploadFile(c, file)
	if err != nil {
		fmt.Println("Failed to upload file:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
		return
	}

	// 5. 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"url": url,
	})
}

// UploadArticleImages 多图上传处理
func (h *UploadHandler) UploadArticleImages(c *gin.Context) {
	// 1. 身份验证
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 2. 解析表单数据
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form data"})
		return
	}

	// 3. 获取所有图片文件
	files := form.File["images"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image files uploaded"})
		return
	}

	// 4. 校验文件数量和单个文件大小
	if len(files) > maxArticleImages {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Too many files, maximum %d allowed", maxArticleImages),
		})
		return
	}

	for _, file := range files {
		if file.Size > 10*1024*1024 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File too large (max 10MB per file)"})
			return
		}
	}

	// 5. 批量上传文件
	urls, err := h.svc.UploadFiles(c, files, maxArticleImages)
	if err != nil {
		fmt.Println("Failed to upload files:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload files"})
		return
	}

	// 6. 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"urls": urls,
	})
}
