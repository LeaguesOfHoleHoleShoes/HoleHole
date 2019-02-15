package util

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"time"
)

func TestStringifyNil(t *testing.T) {
	assert.Equal(t, "null", StringifyJson(nil))
}

func TestTimer(t *testing.T) {
	timer := time.NewTimer(time.Millisecond)
	timer.Stop()
	select {
	case <- timer.C:
		t.Fatal("timer stopped")
	case <- time.After(2 * time.Millisecond):
	}
	timer.Reset(time.Millisecond)
	select {
	case <- timer.C:
	case <- time.After(2 * time.Millisecond):
		t.Fatal("timer reset not work")
	}
}

type inter interface {
	x() int
}
type obj struct {}

func (o *obj) x() int {
	return 1
}

func TestInterfaceSliceCopy(t *testing.T) {
	objs := []*obj{ {}, {}, {}, {} }
	to := make([]inter, len(objs))
	InterfaceSliceCopy(to, objs)

	assert.Len(t, to, 4)
	for _, x := range to {
		assert.Equal(t, 1, x.x())
	}
}