package core

import (
	"testing"
)

func TestOnFkPokers(t *testing.T) {
	ph := PokerHeap{}
	ph.onInit()
	pokers := ph.onFkPokers(2)
	if len(pokers) != 2 {
		t.Fatal("应该为2")
	}
	if len(ph.Pokers) != len(originPokers) - 2 {
		t.Fatal("牌堆长度不对")
	}
}
