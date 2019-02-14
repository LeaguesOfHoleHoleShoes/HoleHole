package model

type BetInfo struct {
	UserAddress string `json:"user_address"`
	Amount uint64 `json:"amount"`
	BlockHeight uint64 `json:"block_height"`
	TxID string `json:"tx_id"`
	Round uint64 `json:"round"`
	// 0为小，1为大
	BetOn int `json:"bet_on"`
}
