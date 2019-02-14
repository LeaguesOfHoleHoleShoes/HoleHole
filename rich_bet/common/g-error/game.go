package g_error

import "errors"

var (
	ErrShouldNotRewardAtHeight = errors.New("shouldn't reward at this height")
	ErrBetAmountTooBig = errors.New("bet amount too big")
	ErrBetBeforeCurRound = errors.New("can't bet before current round")
)
