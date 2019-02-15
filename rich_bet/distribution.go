package rich_bet

import (
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/rich_bet/model"
	"strconv"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/common/log"
	"go.uber.org/zap"
)

// 执行分发
func DoDistribute(config distributionConfig) ([]model.Reward, model.Jackpot) {
	return NewDistribution(config).Distribute()
}

func NewDistribution(config distributionConfig) *distribution {
	return &distribution{ distributionConfig: config }
}

type distributionConfig struct {
	Round uint64
	ResultHash string
	// 平台的地址，用于收取服务费
	PlatformAddress string
	Bets []model.BetInfo
	Jackpot model.Jackpot
}

// 奖励分配。输入：该轮所有用户的押注信息、开奖结果。输出：所有用户的Reward信息
type distribution struct {
	distributionConfig
}

// 执行分配
func (d *distribution) Distribute() (rewards []model.Reward, jackpot model.Jackpot) {
	// 如果后边算出结果了会再次赋值，如果中途报错则应该保留上一次的结果
	jackpot = d.Jackpot
	// 看有没人下注，没有则直接退出
	if len(d.Bets) == 0 {
		return
	}

	// 计算结果
	x, err := strconv.ParseInt(d.ResultHash[len(d.ResultHash) - 1:], 16, 8)
	if err != nil {
		log.L.Error("parse result hash failed", zap.Error(err))
		return
	}
	result := 1
	if x < 8 {
		result = 0
	}

	// 计算各个金额
	var totalBet uint64
	var lossBet uint64
	for _, b := range d.Bets {
		totalBet += b.Amount
		if b.BetOn != result {
			lossBet += b.Amount
		}
	}

	// 2% 平台收取
	platformReward := lossBet * 2 / 100
	rewards = append(rewards, model.Reward{ UserAddress: d.PlatformAddress, Amount: platformReward, Round: d.Round })

	// 98% + 奖金池的奖金
	remainReward := lossBet - platformReward + d.Jackpot.Amount
	remainForJackpot := remainReward
	for _, b := range d.Bets {
		// 用户下注占总数的比例，来赢取输家的98%筹码
		uReward := b.Amount * remainReward / totalBet
		// 如果是赢家则需要+回本金
		if b.BetOn == result {
			uReward += b.Amount
		}
		remainForJackpot -= uReward
		rewards = append(rewards, model.Reward{ UserAddress: b.UserAddress, Amount: uReward, Round: d.Round })
	}

	jackpot.Amount = remainForJackpot

	return
}
