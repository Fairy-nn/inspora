package config

// ElasticSearchConfig ES 配置
type ElasticSearchConfig struct {
	// Addresses ES 节点地址
	Addresses []string `json:"addresses"`
	// Username ES 用户名
	Username string `json:"username"`
	// Password ES 密码
	Password string `json:"password"`
	// MaxRetries 最大重试次数
	MaxRetries int `json:"max_retries"`
	// RequestTimeout 请求超时时间，单位：秒
	RequestTimeout int `json:"request_timeout"`
	// IndexPrefix 索引前缀，不同环境可以使用不同前缀
	IndexPrefix string `json:"index_prefix"`
}