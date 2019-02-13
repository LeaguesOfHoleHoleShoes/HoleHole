package g_error

import "errors"

var (
	ErrShouldNotRewardAtHeight = errors.New("shouldn't reward at this height")
)
