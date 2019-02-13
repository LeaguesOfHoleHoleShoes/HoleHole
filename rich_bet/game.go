package rich_bet

import "github.com/LeaguesOfHoleHoleShoes/HoleHole/rich_bet/common/g-error"

type Database interface {
}

func NewGame() *Game {
	return &Game{}
}

type GameConfig struct {
	// 轮数间距
	Margin uint64
	// 每轮结束下注距开奖块的距离
	EndBetMargin uint64
}

type Game struct {
	GameConfig

	db Database
}

// 用户下注，记录该笔下注信息，并给他本轮的下注总额+amount
func (g *Game) Bet(uAddr string, amount uint64, blockHeight uint64, txID string) error {

	return nil
}

// 触发分发奖励
func (g *Game) Reward(blockHeight uint64) error {
	// 判断该高度是否应该分奖励
	if !shouldReward(blockHeight, g.Margin) {
		return g_error.ErrShouldNotRewardAtHeight
	}

	return nil
}

func shouldReward(blockHeight uint64, margin uint64) bool {
	if blockHeight == 0 {
		return false
	}
	if blockHeight % margin == 0 {
		return true
	}
	return false
}

// 根据height计算round
func roundByHeight(blockHeight uint64, margin uint64, endBetMargin uint64) uint64 {
	return (blockHeight + endBetMargin) / margin
}