package rich_bet

import (
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/rich_bet/model"
	"gopkg.in/mgo.v2"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/common/mongo"
	"gopkg.in/mgo.v2/bson"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/common/log"
	"go.uber.org/zap"
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

func (db *GameDBByMongo) SaveDistribution(rewards []model.Reward, jackpot model.Jackpot) error {
	rwLen := len(rewards)
	objs := make([]interface{}, rwLen)
	txIDs := make([]string, rwLen)
	for i, r := range rewards {
		objs[i] = r
		txIDs[i] = r.TxID
	}

	rwCollection := db.getDB().C(db.rewardTN)
	if err := rwCollection.Insert(objs...); err != nil {
		// 插入失败则删除已插入的数据
		if _, dErr := rwCollection.RemoveAll(bson.M{"txid": bson.M{"$in": txIDs}}); dErr != nil {
			log.L.Error("remove insert failed rewards failed", zap.Error(dErr), zap.Error(err))
		}
		return err
	}
	//log.L.Debug("update jackpot", zap.Uint64("amount", jackpot.Amount))
	return db.getDB().C(db.jackpotTN).Update(bson.M{"tag": jackpot.Tag}, jackpot)
}

// round必须，userAddr如果传空则不带该条件，hasBeenDrawing传<0为false、0为不带该条件、>0为true
func (db *GameDBByMongo) GetRewardsByRound(round uint64, userAddr string, hasBeenDrawing int) (rewards []model.Reward) {
	query := bson.M{"round": round}
	if userAddr != "" {
		query["useraddress"] = userAddr
	}
	if hasBeenDrawing > 0 {
		query["hasbeendrawing"] = true
	} else if hasBeenDrawing < 0 {
		query["hasbeendrawing"] = false
	}

	db.getDB().C(db.rewardTN).Find(query).All(&rewards)
	return
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
	db.getDB().C(db.rewardTN).EnsureIndex(mgo.Index{ Key: []string{"round"} })

	// init jackpot
	jcCollection := db.getDB().C(db.jackpotTN)
	if jc, err := jcCollection.Count(); err != nil {
		panic(err)
	} else {
		if jc == 0 {
			if err = jcCollection.Insert(model.Jackpot{}); err != nil {
				panic(err)
			}
		} else if jc > 1 {
			panic("too many jackpot in db")
		}
	}
}

func (db *GameDBByMongo) ClearTestData() {
	mongo.ClearAllData(db.config, db.dbName)
}