package domain

import "time"

type Article struct {
	ID      int64
	Title   string
	Content string
	Author  Author
	Status  ArticleStatus // 文章状态
	Ctime   time.Time     // 创建时间
	Utime   time.Time     // 更新时间
	ImgUrls []string // 图片地址
}

type Author struct {
	ID   int64
	Name string
}

// 在这里定义文章状态的常量
type ArticleStatus uint8

const (
	ArticleStatusUnknown   ArticleStatus = iota // 未知状态
	ArticleStatusDraft                          // 草稿状态
	ArticleStatusPublished                      // 已发布状态
	ArticleStatusPrivate                        // 私密状态
)

func (a ArticleStatus) ToUint8() uint8 {
	return uint8(a)
}

// 判断文章状态是否有效
func (a ArticleStatus) Valid() bool {
	switch a {
	case ArticleStatusUnknown, ArticleStatusDraft, ArticleStatusPublished, ArticleStatusPrivate:
		return true
	}
	return false
}

// 将状态转换为字符串
func (a ArticleStatus) String() string {
	switch a {
	case ArticleStatusDraft:
		return "草稿状态"
	case ArticleStatusPublished:
		return "已发布状态"
	case ArticleStatusPrivate:
		return "私密状态"
	default:
		return "未知状态"
	}
}

// 生成文章摘要
func (a *Article) GenerateAbstract() string {
	// 文章内容转换为 rune 切片,因为中文字符可能会占用多个字节
	content := []rune(a.Content)
	// 如果内容长度大于 100 个字符，则截取前 100 个字符并添加省略号
	if len(content) > 100 {
		return string(content[:100]) + "..."
	}
	return string(content)
}
