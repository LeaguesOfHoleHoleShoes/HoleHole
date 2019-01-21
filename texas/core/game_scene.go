package core

import "github.com/LeaguesOfHoleHoleShoes/HoleHole/texas/abstracts"

type gameSceneMsg struct {
	uid string
	resultChan chan *abstracts.GameScene
}

func (g *Game) GetScene(uid string) *abstracts.GameScene {
	if g.stopChan == nil {
		return nil
	}

	resultChan := make(chan *abstracts.GameScene)
	g.gameSceneChan <- gameSceneMsg{ uid: uid, resultChan: resultChan }
	return <-resultChan
}

/*
1. 椅子情况：玩家信息，玩家剩余筹码数，自己的手牌，当前D，当前该谁出牌，每个位置是弃牌、all in、正常状态
1. 公共牌
1. 筹码池
*/
func (g *Game) doGetScene(msg gameSceneMsg) {
	result := &abstracts.GameScene{
		CurBet:    g.players[g.curBetPlayer].ID(),
		Players:   map[string]*abstracts.PlayerScene{},
		ChipPools: toChipPoolScene(g.chipPool),
	}

	for _, p := range g.players {
		result.Players[p.ID()] = toPlayerScene(msg.uid, p)
	}

	for _, poker := range g.commonPokers {
		result.CommonPokers = append(result.CommonPokers, &abstracts.PokerScene{ Whole: poker.GetWhole() })
	}

	msg.resultChan <- result
}

func toChipPoolScene(pool *termChipPool) []*abstracts.ChipPoolScene {
	var result []*abstracts.ChipPoolScene
	for next := pool.pool; next != nil; next = next.nextPool {
		result = append(result, &abstracts.ChipPoolScene{ Chips: next.totalChip() })
	}
	return result
}

func toPlayerScene(uid string, p abstracts.Player) *abstracts.PlayerScene {
	rp := &abstracts.PlayerScene{
		UserID: p.ID(),
		RemainChip: p.RemainChip(),
	}
	if p.Discarded() {
		rp.Status = abstracts.PlayerStatusDiscarded
	}
	if p.AllInned() {
		rp.Status = abstracts.PlayerStatusAllInned
	}

	// 用户只可见自己的手牌
	if p.ID() == uid {
		pokers := p.Pokers()
		for _, poker := range pokers {
			rp.Pokers = append(rp.Pokers, &abstracts.PokerScene{ Whole: poker.GetWhole() })
		}
	}
	return rp
}