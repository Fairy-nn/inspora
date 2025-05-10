package domain

// 打赏的目标
type Target struct {
	Biz     string
	BizId   int64
	BizName string
	UserID  int64
}
// 打赏的金额
type Reward struct {
	ID     int64
	UserID int64
	Target Target
	Amt    int64
	Status RewardStatus
	// 这里未来可以引入货币类型
}
// 打赏的状态
type RewardStatus int
const (
	RewardStatusUnknown RewardStatus = iota
	RewardStatusInit
	RewardStatusPaid
	RewardStatusFailed
)
// 打赏的状态转换为uint8
func (s Reward) Completed() bool {
	return s.Status == RewardStatusFailed || s.Status == RewardStatusPaid
}

// 二维码
type CodeURL struct {
	Rid int64
	URL string
}
