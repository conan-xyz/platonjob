package conf

// Config ...
type Config struct {
	ChainID          int64   `json:"chain_id" yaml:"chainId"`
	Async            *bool   `json:"async" yaml:"async"`
	RawURL           string  `json:"raw_url" yaml:"rawURL"`
	Arp              string  `json:"arp" yaml:"arp"`
	RewardBlock      int64   `json:"reward_block" yaml:"rewardBlock"`
	DelegateBlock    int64   `json:"delegate_block" yaml:"delegateBlock"`
	Addrs            []Addr  `json:"addrs" yaml:"addrs"`
	DstAddr          string  `json:"dst_addr" yaml:"dstAddr"`
	MinDelegate      float64 `json:"min_delegate" yaml:"minDelegate"`
	RewardGasLimit   uint64  `json:"reward_gas_limit" yaml:"rewardGasLimit"`
	DelegateGasLimit uint64  `json:"delegate_gas_limit" yaml:"delegateGasLimit"`
}

// Addr ...
type Addr struct {
	PrivateKey string `json:"private_key" yaml:"privateKey"`
	NodeID     string `json:"node_id" yaml:"nodeId"`
}
