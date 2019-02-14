package model

type Reward struct {
	UserAddress string `json:"user_address"`
	Amount uint64 `json:"amount"`
	Round uint64 `json:"round"`
	// 是否已经提款
	HasBeenDrawing bool `json:"has_been_drawing"`
}
