package core

import (
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/texas/abstracts"
)

type HMatcher struct {}

// h1 > h2 return 1, h1 < h2 return -1, h1 == h2 return 0
func (hm *HMatcher) Cmp(h1, h2 abstracts.Hand) int {
	// 如果牌型一样才比较权重，type越大代表牌型越大
	if h1.HandType() == h2.HandType() {
		// 比较weight
		if h1.Weight() == h2.Weight() {
			return 0
		}else if h1.Weight() > h2.Weight() {
			return 1
		}else {
			return -1
		}
	}else if h1.HandType() > h2.HandType() {
		return 1
	}else {
		return -1
	}
}

