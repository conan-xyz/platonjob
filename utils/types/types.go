package types

// RewardInfo ...
type RewardInfo struct {
	NodeID     string `json:"nodeID"`
	Reward     string `json:"reward"`
	StakingNum uint64 `json:"stakingNum"`
}

// RewardRes ...
type RewardResponse struct {
	Code int64         `json:"code"`
	Ret  []*RewardInfo `json:"ret"`
}
