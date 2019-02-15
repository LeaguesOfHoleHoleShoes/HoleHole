package rich_bet

import (
	"testing"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/rich_bet/model"
	"github.com/stretchr/testify/assert"
	"gopkg.in/check.v1"
)

const(
	testDBName = "game_db_test"
)

var _ = check.Suite(&GameDBByMongoSuite{})

func Test(t *testing.T) { check.TestingT(t) }

type GameDBByMongoSuite struct {
	gDB *GameDBByMongo
}

// suite 开始时初始化
func (s *GameDBByMongoSuite) SetUpSuite(c *check.C) {}

// suite 结束时做的事
func (s *GameDBByMongoSuite) TearDownSuite(c *check.C) {}

// 每一个test case 的开始初始化
func (s *GameDBByMongoSuite) SetUpTest(c *check.C) {
	s.gDB = NewGameDBByMongo([]string{"localhost"}, testDBName)
}

// 每一个test case 的结束是做的事
func (s *GameDBByMongoSuite) TearDownTest(c *check.C) {
	s.gDB.ClearTestData()
}

func (s *GameDBByMongoSuite) TestGameDBByMongo_GetBetsByRound(t *check.C) {
	err := s.gDB.DoBet(model.BetInfo{ UserAddress: "0x123", Amount: 321, BlockHeight: 1, TxID: "123", Round: 1, BetOn: 0 })
	assert.NoError(t, err)
	err = s.gDB.DoBet(model.BetInfo{ UserAddress: "0x123", Amount: 321, BlockHeight: 1, TxID: "1231", Round: 1, BetOn: 0 })
	assert.NoError(t, err)
	err = s.gDB.DoBet(model.BetInfo{ UserAddress: "0x123", Amount: 321, BlockHeight: 1, TxID: "1232", Round: 1, BetOn: 0 })
	assert.NoError(t, err)

	err = s.gDB.DoBet(model.BetInfo{ UserAddress: "0x123", Amount: 321, BlockHeight: 1, TxID: "1233", Round: 2, BetOn: 0 })
	assert.NoError(t, err)

	bets := s.gDB.GetBetsByRound(1)
	assert.Len(t, bets, 3)

	bets = s.gDB.GetBetsByRound(2)
	assert.Len(t, bets, 1)
}

func (s *GameDBByMongoSuite) TestGameDBByMongo_DoBet(t *check.C) {
	err := s.gDB.DoBet(model.BetInfo{ UserAddress: "0x123", Amount: 321, BlockHeight: 1, TxID: "123", Round: 1, BetOn: 0 })
	assert.NoError(t, err)

	err = s.gDB.DoBet(model.BetInfo{ UserAddress: "0x123", Amount: 321, BlockHeight: 1, TxID: "123", Round: 1, BetOn: 0 })
	assert.Error(t, err)
}

func (s *GameDBByMongoSuite) TestGameDBByMongo_SaveDistribution(t *check.C) {
	jp := s.gDB.GetJackpot()
	assert.Equal(t, 0, jp.Tag)
	assert.Equal(t, uint64(0), jp.Amount)

	jp.Amount = 22
	// 交易ID重复会导致插入失败，数据库中应该没有任何改动才对
	err := s.gDB.SaveDistribution([]model.Reward{
		{ TxID: "0x123", UserAddress: "0x123", Amount: 11, Round: 1 },
		{ TxID: "0x123", UserAddress: "0x123", Amount: 11, Round: 1 },
		{ TxID: "0x123", UserAddress: "0x123", Amount: 11, Round: 1 },
		{ TxID: "0x123", UserAddress: "0x123", Amount: 11, Round: 2 },
	}, jp)
	assert.Error(t, err)

	rws := s.gDB.GetRewardsByRound(1, "", 0)
	assert.Len(t, rws, 0)
	rws = s.gDB.GetRewardsByRound(2, "", 0)
	assert.Len(t, rws, 0)

	jp = s.gDB.GetJackpot()
	assert.Equal(t, 0, jp.Tag)
	assert.Equal(t, uint64(0), jp.Amount)

	jp.Amount = 22
	// 正确的插入
	err = s.gDB.SaveDistribution([]model.Reward{
		{ TxID: "0x123", UserAddress: "0x123", Amount: 11, Round: 1 },
		{ TxID: "0x1231", UserAddress: "0x123", Amount: 11, Round: 1 },
		{ TxID: "0x1232", UserAddress: "0x123", Amount: 11, Round: 1 },
		{ TxID: "0x1233", UserAddress: "0x123", Amount: 11, Round: 2 },
	}, jp)
	assert.NoError(t, err)

	rws = s.gDB.GetRewardsByRound(1, "", 0)
	assert.Len(t, rws, 3)
	rws = s.gDB.GetRewardsByRound(2, "", 0)
	assert.Len(t, rws, 1)

	jp = s.gDB.GetJackpot()
	assert.Equal(t, 0, jp.Tag)
	assert.Equal(t, uint64(22), jp.Amount)
}

func (s *GameDBByMongoSuite) TestGameDBByMongo_SaveInvalidBet(t *check.C) {
	err := s.gDB.SaveInvalidBet(model.InvalidBet{InvalidType: model.InvalidAmount})
	assert.NoError(t, err)
	n, err := s.gDB.getDB().C(s.gDB.invalidBetTN).Count()
	assert.NoError(t, err)
	assert.Equal(t, 1, n)
}
