package config

// OSSConfig OSS配置
type OSSConfig struct {
	// OSS类型，用于指定连接的地域节点
    Endpoint        string `yaml:"endpoint"`
    // 访问密钥ID
	AccessKeyID     string `yaml:"access_key_id"`
    // 访问密钥Secret
	// 用于身份验证和授权
	AccessKeySecret string `yaml:"access_key_secret"`
    // 存放文件的逻辑容器，每个用户可以有多个 Bucket。
	// 其实就是相当于一个文件夹
	BucketName      string `yaml:"bucket_name"`
    // 文件访问的基础 URL
	BaseURL         string `yaml:"base_url"`
}