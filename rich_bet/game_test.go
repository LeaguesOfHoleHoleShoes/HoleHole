package rich_bet

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestShouldReward(t *testing.T) {
	assert.False(t, shouldReward(0, 10))
	assert.False(t, shouldReward(9, 10))
	assert.False(t, shouldReward(11, 10))
	assert.True(t, shouldReward(10, 10))
	assert.True(t, shouldReward(20, 10))
	assert.False(t, shouldReward(21, 10))
}

func TestRoundByHeight(t *testing.T) {
	assert.Equal(t, uint64(0), roundByHeight(1, 10, 5))
	assert.Equal(t, uint64(0), roundByHeight(4, 10, 5))

	assert.Equal(t, uint64(1), roundByHeight(5, 10, 5))
	assert.Equal(t, uint64(1), roundByHeight(6, 10, 5))
	assert.Equal(t, uint64(1), roundByHeight(14, 10, 5))

	assert.Equal(t, uint64(2), roundByHeight(15, 10, 5))
	assert.Equal(t, uint64(2), roundByHeight(16, 10, 5))
	assert.Equal(t, uint64(3), roundByHeight(25, 10, 5))
}