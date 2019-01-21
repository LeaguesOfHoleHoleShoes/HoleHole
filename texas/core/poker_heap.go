package core

import (
	"go.uber.org/zap"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/texas/abstracts"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/util"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/texas/core/hand_processor"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/log"
)

func newPokerHeap() *PokerHeap {
	ph := &PokerHeap{}
	ph.onInit()
	return ph
}

// todo 测试多协程是否有问题，应该没问题，因为不会对该数组做修改操作，都是读操作
// 最原始的牌，所有桌子的牌都由该牌组随机排序后生成的
var originPokers = hand_processor.MakeDeckOfCards(false)

// 牌堆（每次新建game都会重新创建该对象）
type PokerHeap struct {
	// 牌堆里的牌
	Pokers []*hand_processor.Poker `json:"pokers"`
}

func (pokerHeap *PokerHeap) DispatchPokers(count int) []abstracts.Poker {
	ps := pokerHeap.onFkPokers(count)
	result := make([]abstracts.Poker, len(ps))
	util.InterfaceSliceCopy(result, ps)
	return result
}

func (pokerHeap *PokerHeap) onInit() {
	log.L.Debug("牌堆初始化")
	// 洗牌
	pokerHeap.shuffleTheDeck()
}

// 洗牌
func (pokerHeap *PokerHeap) shuffleTheDeck() {
	totalLen := len(originPokers)
	pokerHeap.Pokers = append([]*hand_processor.Poker{}, originPokers...)
	for i := range pokerHeap.Pokers {
		j := util.RandANum(totalLen)
		pokerHeap.Pokers[i], pokerHeap.Pokers[j] = pokerHeap.Pokers[j], pokerHeap.Pokers[i]
	}
}

// 从牌堆取牌来发
func (pokerHeap *PokerHeap) onFkPokers(pieces int) (result []*hand_processor.Poker) {
	log.L.Debug("onFkPokers", zap.Int("count", pieces), zap.Int("heap len", len(pokerHeap.Pokers)))
	result = pokerHeap.Pokers[0:pieces]
	pokerHeap.Pokers = pokerHeap.Pokers[pieces:]
	return
}