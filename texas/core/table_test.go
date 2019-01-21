package core

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/texas/abstracts"
)

func TestNilSeat(t *testing.T) {
	table := &Table{ seats: make([]abstracts.User, 5) }
	assert.True(t, table.seats[2] == nil)
}
