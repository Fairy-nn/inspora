package domain

// FollowRelation 关注数据
type FollowRelation struct {
	// 被关注的人
	Followee int64
	// 关注的人
	Follower int64
}

// FollowStatistics 关注统计
type FollowStatistics struct {
	// 被多少人关注
	Followers int64
	// 自己关注了多少人
	Followees int64
}
