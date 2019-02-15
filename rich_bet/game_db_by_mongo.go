package rich_bet

import (
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/rich_bet/model"
	"gopkg.in/mgo.v2"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/common/mongo"
	"gopkg.in/mgo.v2/bson"
)

func NewGameDBByMongo(hosts []string, dbName string) *GameDBByMongo {
	db := &GameDBByMongo{
		config: mongo.NewDbConfig(hosts),
		dbName: dbName,

		betInfoTN: "bet_info",
		invalidBetTN: "invalid_bet",
		jackpotTN: "jackpot",
		rewardTN: "reward",
	}

	db.migrate()

	return db
}

type GameDBByMongo struct {
	config *mgo.DialInfo
	dbName string

	betInfoTN string
	invalidBetTN string
	jackpotTN string
	rewardTN string
}

// 对txid做unique，则直接Insert即可
func (db *GameDBByMongo) DoBet(info model.BetInfo) error {
	return db.getDB().C(db.betInfoTN).Insert(info)
}

func (db *GameDBByMongo) GetBetsByRound(round uint64) (result []model.BetInfo) {
	db.getDB().C(db.betInfoTN).Find(bson.M{"round": round}).All(&result)
	return
}

// 事务问题
func (db *GameDBByMongo) SaveDistribution(rewards []model.Reward, jackpot model.Jackpot) error {
	if err := db.getDB().C(db.rewardTN).Insert(rewards); err != nil {
		return err
	}
	return db.getDB().C(db.jackpotTN).Update(bson.M{"tag": jackpot.Tag}, jackpot)
}

func (db *GameDBByMongo) GetJackpot() (result model.Jackpot) {
	db.getDB().C(db.jackpotTN).Find(bson.M{"tag": result.Tag}).One(&result)
	return
}

func (db *GameDBByMongo) SaveInvalidBet(bet model.InvalidBet) error {
	return db.getDB().C(db.invalidBetTN).Insert(bet)
}

func (db *GameDBByMongo) getDB() *mgo.Database {
	return mongo.GetDB(db.config).DB(db.dbName)
}

func (db *GameDBByMongo) migrate() {
	db.getDB().C(db.betInfoTN).EnsureIndex(mgo.Index{ Key: []string{"txid"}, Unique: true })
	db.getDB().C(db.betInfoTN).EnsureIndex(mgo.Index{ Key: []string{"round"} })

	db.getDB().C(db.invalidBetTN).EnsureIndex(mgo.Index{ Key: []string{"txid"}, Unique: true })

	db.getDB().C(db.rewardTN).EnsureIndex(mgo.Index{ Key: []string{"txid"}, Unique: true })
}