package model

type Jackpot struct {
	// 数据库中做唯一标识0，只有一条该数据
	Tag int `json:"tag"`
	Amount uint64 `json:"amount"`
}