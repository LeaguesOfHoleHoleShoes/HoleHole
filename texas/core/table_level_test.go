package core

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestTableLevel(t *testing.T) {
	l := TableLevels[4]
	assert.Equal(t, 0, int(l.Xm))
}
