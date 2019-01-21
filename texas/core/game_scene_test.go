package core

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"time"
	"go.uber.org/zap"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/texas/abstracts"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/log"
)

func TestGame_GetScene(t *testing.T) {
	resultC := make(chan *GameResult)
	g := NewGame(10, newFakePlayersInTable1(), &fakeMsgSender{}, resultC)
	go g.Run()
	time.Sleep(100 * time.Millisecond)

	// 小盲10，大盲20
	g.betRight(t, 1, 10)
	g.betRight(t, 2, 20)
	// 第一轮下注
	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfBet, 100))
	g.OnMsg(g.newPlayerActionMsg(4, abstracts.GameActionOfBet, 100))
	g.OnMsg(g.newPlayerActionMsg(0, abstracts.GameActionOfBet, 100))
	gScene := g.GetScene("1")
	assert.Equal(t, "1", gScene.CurBet)
	assert.Len(t, gScene.Players["1"].Pokers, 2)
	assert.Nil(t, gScene.Players["0"].Pokers)
	assert.Equal(t, 1980, int(gScene.Players["2"].RemainChip))
	g.OnMsg(g.newPlayerActionMsg(1, abstracts.GameActionOfBet, 100))
	g.OnMsg(g.newPlayerActionMsg(2, abstracts.GameActionOfBet, 100))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, int(g.curRound))
	assert.Equal(t, 3, int(g.curBetPlayer))
	g.betRight(t, 1, 110)
	g.betRight(t, 2, 120)
	g.betRight(t, 3, 100)
	g.betRight(t, 4, 100)
	g.betRight(t, 0, 100)
	// 所有人下注到相同，进入下一轮
	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfBet, 20))
	g.OnMsg(g.newPlayerActionMsg(4, abstracts.GameActionOfBet, 20))
	g.OnMsg(g.newPlayerActionMsg(0, abstracts.GameActionOfBet, 20))
	g.OnMsg(g.newPlayerActionMsg(1, abstracts.GameActionOfBet, 10))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 2, int(g.curRound))
	assert.Equal(t, 1, int(g.curBetPlayer))

	// 第二轮下注，测试过牌逻辑正常
	g.OnMsg(g.newPlayerActionMsg(1, abstracts.GameActionOfBet, 0))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 2, int(g.curRound))
	assert.Equal(t, 2, int(g.curBetPlayer))
	g.OnMsg(g.newPlayerActionMsg(2, abstracts.GameActionOfBet, 0))
	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfBet, 0))
	g.OnMsg(g.newPlayerActionMsg(4, abstracts.GameActionOfBet, 0))
	g.OnMsg(g.newPlayerActionMsg(0, abstracts.GameActionOfBet, 0))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 3, int(g.curRound))
	assert.Equal(t, 1, int(g.curBetPlayer))
	// 第三轮下注
	g.OnMsg(g.newPlayerActionMsg(1, abstracts.GameActionOfBet, 300))
	g.OnMsg(g.newPlayerActionMsg(2, abstracts.GameActionOfBet, 300))
	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfBet, 300))
	g.OnMsg(g.newPlayerActionMsg(4, abstracts.GameActionOfBet, 300))
	g.OnMsg(g.newPlayerActionMsg(0, abstracts.GameActionOfBet, 300))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 4, int(g.curRound))
	assert.Equal(t, 1, int(g.curBetPlayer))
	// 第四轮下注（最后一轮）
	g.OnMsg(g.newPlayerActionMsg(1, abstracts.GameActionOfBet, 300))
	g.OnMsg(g.newPlayerActionMsg(2, abstracts.GameActionOfBet, 300))
	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfBet, 300))
	//gScene = g.GetScene("1")
	//log.L.Debug(util.StringifyJson(gScene))
	g.OnMsg(g.newPlayerActionMsg(4, abstracts.GameActionOfBet, 300))
	g.OnMsg(g.newPlayerActionMsg(0, abstracts.GameActionOfBet, 300))
	time.Sleep(100 * time.Millisecond)
	g.betRight(t, 1, 720)
	g.betRight(t, 2, 720)
	g.betRight(t, 3, 720)
	g.betRight(t, 4, 720)
	g.betRight(t, 0, 720)
	assert.Equal(t, 5, int(g.curRound))

	result := <- resultC
	assert.Equal(t, g.id, result.id)
	assert.Len(t, result.players, 5)

	for i, pr := range result.players {
		change, isAdd := pr.Result()
		log.L.Debug("player result", zap.Uint("player", i), zap.Uint64("result", change), zap.Bool("is add", isAdd))
	}
}
