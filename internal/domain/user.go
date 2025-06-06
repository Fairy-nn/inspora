package domain

// User领域对象，是DDD中的entity，表示一个用户
// 领域对象是业务逻辑的核心，包含了业务规则和行为
type User struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Username string `json:"username"`
	Ctime    int64  `json:"ctime"` // 创建时间
	Phone    string `json:"phone"`
	Name     string `json:"name"`
	Balance  int64  `json:"balance"` // 用户余额，单位：分
	// Utime   int64  `json:"utime"` // 更新时间
}
