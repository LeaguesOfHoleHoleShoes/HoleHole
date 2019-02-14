package rich_bet

import (
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/rich_bet/common/g-error"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/rich_bet/model"
	"sync"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/log"
	"go.uber.org/zap"
)

type GameDB interface {
	DoBet(info model.BetInfo) error
	GetBetsByRound(round uint64) []model.BetInfo
	SaveDistribution(rewards []model.Reward, jackpot model.Jackpot) error
	GetJackpot() model.Jackpot
	SaveInvalidBet(bet model.InvalidBet) error
	//SaveRewards(rewards []model.Reward) error
	//UpdateJackpot(jackpot model.Jackpot) error
}

type ChainDB interface {
	CurrentHeight() uint64
}

func NewGame(config GameConfig) *Game {
	return &Game{ GameConfig: config }
}

type GameConfig struct {
	// 轮数间距
	Margin uint64
	// 每轮结束下注距开奖块的距离
	EndBetMargin uint64
	// 平台的地址，用于收取服务费
	PlatformAddress string
	// 最多下多少注（uint64按9位小数记则只能支持到184亿）
	MaxBet uint64
	// 最少下注数
	MinBet uint64

	ChainDB ChainDB
}

type Game struct {
	GameConfig

	gDB GameDB

	rewardLock sync.Mutex
}

// 用户下注，记录该笔下注信息，并给他本轮的下注总额+amount
func (g *Game) Bet(info model.BetInfo) error {
	// 不能对小于当前轮下注
	round := roundByHeight(info.BlockHeight, g.Margin, g.EndBetMargin)
	curRound := roundByHeight(g.ChainDB.CurrentHeight(), g.Margin, g.EndBetMargin)
	if round < curRound {
		addInvalidBet(model.InvalidBet{ BetInfo: info, InvalidType: model.InvalidRound, }, g.gDB)
		return g_error.ErrBetBeforeCurRound
	}
	info.Round = round

	// 要做记录，到时审核退回
	if info.Amount > g.MaxBet || info.Amount < g.MinBet {
		addInvalidBet(model.InvalidBet{ BetInfo: info, InvalidType: model.InvalidAmount, }, g.gDB)
		return g_error.ErrBetAmountTooBig
	}

	return g.gDB.DoBet(info)
}

func addInvalidBet(ib model.InvalidBet, gDB GameDB) {
	if err := gDB.SaveInvalidBet(ib); err != nil {
		log.L.Error("save invalid bet failed", zap.String("user addr", ib.UserAddress), zap.Uint64("amount", ib.Amount), zap.String("tx id", ib.TxID), zap.Int("invalid type", ib.InvalidType))
	}
}

// 触发分发奖励
// 同时刻只能有一个Reward程序在跑。因此可能部署一个Reward程序，多个bet程序，但只是可能。
// todo 要考虑如果该节点落后会出现什么情况
func (g *Game) Reward(blockHeight uint64, blockHash string) error {
	g.rewardLock.Lock()
	defer g.rewardLock.Unlock()

	// 判断该高度是否应该分奖励
	if !shouldReward(blockHeight, g.Margin) {
		return g_error.ErrShouldNotRewardAtHeight
	}

	// 当前块已经是下一轮了，因此要-1
	round := roundByHeight(blockHeight, g.Margin, g.EndBetMargin) - 1
	bets := g.gDB.GetBetsByRound(round)
	// todo 判断是否已经分配过

	// do distribute
	rewards, jackpot := DoDistribute(distributionConfig{
		Round: round,
		ResultHash: blockHash,
		PlatformAddress: g.PlatformAddress,
		Bets: bets,
		Jackpot: g.gDB.GetJackpot(),
	})

	// save rewards
	return g.gDB.SaveDistribution(rewards, jackpot)
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