package rich_bet

import (
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/rich_bet/model"
)

type GameDBByMongo struct {

}

func (db *GameDBByMongo) DoBet(info model.BetInfo) error {
	panic("implement me")
}

func (db *GameDBByMongo) GetBetsByRound(round uint64) []model.BetInfo {
	panic("implement me")
}

func (db *GameDBByMongo) SaveDistribution(rewards []model.Reward, jackpot model.Jackpot) error {
	panic("implement me")
}

func (db *GameDBByMongo) GetJackpot() model.Jackpot {
	panic("implement me")
}

func (db *GameDBByMongo) SaveInvalidBet(bet model.InvalidBet) error {
	panic("implement me")
}
