package model

const (
	InvalidAmount = iota
	InvalidRound
)

type InvalidBet struct {
	BetInfo
	InvalidType int `json:"invalid_type"`
}
