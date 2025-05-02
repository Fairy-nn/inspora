package domain

type Interaction struct {
	ViewCnt    int64 `json:"view_cnt"`
	LikeCnt    int64 `json:"like_cnt"`
	CollectCnt int64 `json:"collect_cnt"`
	Collected  bool  `json:"collected"`
	Liked      bool  `json:"liked"`
}
