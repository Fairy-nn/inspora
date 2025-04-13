package domain

// User领域对象，是DDD中的entity，表示一个用户
// 领域对象是业务逻辑的核心，包含了业务规则和行为
type User struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Username string `json:"username"`
	Id       string `json:"id"`
	Ctime    int64  `json:"ctime"` // 创建时间
}
