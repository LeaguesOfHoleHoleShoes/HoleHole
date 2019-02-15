package core

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"time"
	"go.uber.org/zap"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/texas/abstracts"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/common/log"
)

// 正常下注到结束
func TestGame1(t *testing.T) {
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

// 有人all in有人弃牌正常到结束
func TestGame2(t *testing.T) {
	resultC := make(chan *GameResult)
	// 2是1500，其他人都是2000，最后一轮2 all in，其他人不all in，但是超过1500，触发分池
	g := NewGame(10, newFakePlayersInTable2(), &fakeMsgSender{}, resultC)
	go g.Run()
	time.Sleep(100 * time.Millisecond)

	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfBet, 100))
	g.OnMsg(g.newPlayerActionMsg(4, abstracts.GameActionOfBet, 100))
	g.OnMsg(g.newPlayerActionMsg(0, abstracts.GameActionOfDiscard, 0))
	g.OnMsg(g.newPlayerActionMsg(1, abstracts.GameActionOfDiscard, 0))
	g.OnMsg(g.newPlayerActionMsg(2, abstracts.GameActionOfBet, 80))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 2, int(g.curRound))
	// 1已经弃牌
	assert.Equal(t, 2, int(g.curBetPlayer))

	// 0、1已经弃牌，因此会提示不能操作，但不影响逻辑
	g.OnMsg(g.newPlayerActionMsg(1, abstracts.GameActionOfBet, 0))
	g.OnMsg(g.newPlayerActionMsg(2, abstracts.GameActionOfBet, 0))
	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfBet, 0))
	g.OnMsg(g.newPlayerActionMsg(4, abstracts.GameActionOfBet, 0))
	g.OnMsg(g.newPlayerActionMsg(0, abstracts.GameActionOfBet, 0))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 3, int(g.curRound))
	// 从未弃牌的玩家开始
	assert.Equal(t, 2, int(g.curBetPlayer))

	// 0、1已经弃牌，因此会提示不能操作，但不影响逻辑
	g.OnMsg(g.newPlayerActionMsg(2, abstracts.GameActionOfBet, 0))
	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfBet, 0))
	g.OnMsg(g.newPlayerActionMsg(4, abstracts.GameActionOfBet, 0))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 4, int(g.curRound))
	// 从未弃牌的玩家开始
	assert.Equal(t, 2, int(g.curBetPlayer))

	g.OnMsg(g.newPlayerActionMsg(2, abstracts.GameActionOfBet, 1400))
	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfBet, 1600))
	g.OnMsg(g.newPlayerActionMsg(4, abstracts.GameActionOfBet, 1600))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 5, int(g.curRound))
	// 从未弃牌的玩家开始。2 all in了，因此会从3开始
	assert.Equal(t, 3, int(g.curBetPlayer))

	result := <- resultC
	assert.Equal(t, g.id, result.id)
	assert.Len(t, result.players, 5)

	for i, pr := range result.players {
		change, isAdd := pr.Result()
		log.L.Debug("player result", zap.Uint("player", i), zap.Uint64("result", change), zap.Bool("is add", isAdd))
	}
}

// 只有一个人没有弃牌
func TestGame3(t *testing.T) {
	resultC := make(chan *GameResult)
	// 2是1500，其他人都是2000，最后一轮2 all in，其他人不all in，但是超过1500，触发分池
	g := NewGame(10, newFakePlayersInTable2(), &fakeMsgSender{}, resultC)
	go g.Run()
	time.Sleep(100 * time.Millisecond)

	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfBet, 100))
	g.OnMsg(g.newPlayerActionMsg(4, abstracts.GameActionOfBet, 100))
	g.OnMsg(g.newPlayerActionMsg(0, abstracts.GameActionOfDiscard, 0))
	g.OnMsg(g.newPlayerActionMsg(1, abstracts.GameActionOfDiscard, 0))
	g.OnMsg(g.newPlayerActionMsg(2, abstracts.GameActionOfBet, 80))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 2, int(g.curRound))
	// 1已经弃牌
	assert.Equal(t, 2, int(g.curBetPlayer))

	g.OnMsg(g.newPlayerActionMsg(2, abstracts.GameActionOfDiscard, 0))
	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfDiscard, 0))

	result := <- resultC
	assert.Equal(t, g.id, result.id)
	assert.Len(t, result.players, 5)

	for i, pr := range result.players {
		change, isAdd := pr.Result()
		log.L.Debug("player result", zap.Uint("player", i), zap.Uint64("result", change), zap.Bool("is add", isAdd))
	}
}

// 有人all in，随后弃牌到只剩一个人，发牌到最后结束
func TestGame4(t *testing.T) {
	resultC := make(chan *GameResult)
	// 2是1500，其他人都是2000，最后一轮2 all in，其他人不all in，但是超过1500，触发分池
	g := NewGame(10, newFakePlayersInTable2(), &fakeMsgSender{}, resultC)
	go g.Run()
	time.Sleep(100 * time.Millisecond)

	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfBet, 100))
	g.OnMsg(g.newPlayerActionMsg(4, abstracts.GameActionOfBet, 100))
	g.OnMsg(g.newPlayerActionMsg(0, abstracts.GameActionOfDiscard, 0))
	g.OnMsg(g.newPlayerActionMsg(1, abstracts.GameActionOfDiscard, 0))
	g.OnMsg(g.newPlayerActionMsg(2, abstracts.GameActionOfBet, 80))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 2, int(g.curRound))
	// 1已经弃牌
	assert.Equal(t, 2, int(g.curBetPlayer))

	g.OnMsg(g.newPlayerActionMsg(2, abstracts.GameActionOfBet, 1400))
	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfBet, 1500))
	g.OnMsg(g.newPlayerActionMsg(4, abstracts.GameActionOfDiscard, 0))

	result := <- resultC
	assert.Equal(t, g.id, result.id)
	assert.Len(t, result.players, 5)

	for i, pr := range result.players {
		change, isAdd := pr.Result()
		log.L.Debug("player result", zap.Uint("player", i), zap.Uint64("result", change), zap.Bool("is add", isAdd))
	}
}

// 有人all in、弃牌，测试最后那个人还未操作之前是否就触发了dealingCardsToEnd
func TestGame5(t *testing.T) {
	resultC := make(chan *GameResult)
	// 2是1500，其他人都是2000，最后一轮2 all in，其他人不all in，但是超过1500，触发分池
	g := NewGame(10, newFakePlayersInTable2(), &fakeMsgSender{}, resultC)
	go g.Run()
	time.Sleep(100 * time.Millisecond)

	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfBet, 100))
	g.OnMsg(g.newPlayerActionMsg(4, abstracts.GameActionOfBet, 100))
	g.OnMsg(g.newPlayerActionMsg(0, abstracts.GameActionOfDiscard, 0))
	g.OnMsg(g.newPlayerActionMsg(1, abstracts.GameActionOfDiscard, 0))
	g.OnMsg(g.newPlayerActionMsg(2, abstracts.GameActionOfBet, 80))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 2, int(g.curRound))
	// 1已经弃牌
	assert.Equal(t, 2, int(g.curBetPlayer))

	g.OnMsg(g.newPlayerActionMsg(2, abstracts.GameActionOfBet, 1400))
	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfDiscard, 0))
	g.OnMsg(g.newPlayerActionMsg(4, abstracts.GameActionOfBet, 1500))

	log.L.Debug("wait game result")
	result := <- resultC
	assert.Equal(t, g.id, result.id)
	assert.Len(t, result.players, 5)

	for i, pr := range result.players {
		change, isAdd := pr.Result()
		log.L.Debug("player result", zap.Uint("player", i), zap.Uint64("result", change), zap.Bool("is add", isAdd))
	}
}

// 超时顺位
func TestGame6(t *testing.T) {
	// 修改超时计时时间
	betTimeout = 200 * time.Millisecond
	resultC := make(chan *GameResult)
	// 2是1500，其他人都是2000，最后一轮2 all in，其他人不all in，但是超过1500，触发分池
	g := NewGame(10, newFakePlayersInTable2(), &fakeMsgSender{}, resultC)
	go g.Run()
	time.Sleep(50 * time.Millisecond)

	// 做操作会刷新超时时间
	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfBet, 100))
	g.OnMsg(g.newPlayerActionMsg(4, abstracts.GameActionOfBet, 100))
	g.OnMsg(g.newPlayerActionMsg(0, abstracts.GameActionOfDiscard, 0))
	g.OnMsg(g.newPlayerActionMsg(1, abstracts.GameActionOfDiscard, 0))
	g.OnMsg(g.newPlayerActionMsg(2, abstracts.GameActionOfBet, 80))
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 2, int(g.curRound))
	assert.Equal(t, 2, int(g.curBetPlayer))

	// 2不做任何操作，其他人过牌
	// 等待2超时，但是3不能超时，上边睡了一个50
	time.Sleep(170 * time.Millisecond)
	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfBet, 0))
	g.OnMsg(g.newPlayerActionMsg(4, abstracts.GameActionOfBet, 0))
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 3, int(g.curRound))
	assert.Equal(t, 2, int(g.curBetPlayer))

	// 等待2超时，上边睡了一个50
	time.Sleep(170 * time.Millisecond)
	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfBet, 0))
	g.OnMsg(g.newPlayerActionMsg(4, abstracts.GameActionOfBet, 0))
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 4, int(g.curRound))
	assert.Equal(t, 2, int(g.curBetPlayer))

	// 等待2超时
	time.Sleep(170 * time.Millisecond)
	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfBet, 0))
	g.OnMsg(g.newPlayerActionMsg(4, abstracts.GameActionOfBet, 0))
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 5, int(g.curRound))
	assert.Equal(t, 2, int(g.curBetPlayer))
	assert.False(t, g.players[2].Discarded())

	result := <- resultC
	assert.Equal(t, g.id, result.id)
	assert.Len(t, result.players, 5)

	for i, pr := range result.players {
		change, isAdd := pr.Result()
		log.L.Debug("player result", zap.Uint("player", i), zap.Uint64("result", change), zap.Bool("is add", isAdd))
	}
}

// 超时弃牌
func TestGame7(t *testing.T) {
	// 修改超时计时时间
	betTimeout = 200 * time.Millisecond
	resultC := make(chan *GameResult)
	// 2是1500，其他人都是2000，最后一轮2 all in，其他人不all in，但是超过1500，触发分池
	g := NewGame(10, newFakePlayersInTable2(), &fakeMsgSender{}, resultC)
	go g.Run()
	time.Sleep(50 * time.Millisecond)

	// 做操作会刷新超时时间
	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfBet, 100))
	g.OnMsg(g.newPlayerActionMsg(4, abstracts.GameActionOfBet, 100))
	g.OnMsg(g.newPlayerActionMsg(0, abstracts.GameActionOfDiscard, 0))
	g.OnMsg(g.newPlayerActionMsg(1, abstracts.GameActionOfDiscard, 0))
	g.OnMsg(g.newPlayerActionMsg(2, abstracts.GameActionOfBet, 80))
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 2, int(g.curRound))
	assert.Equal(t, 2, int(g.curBetPlayer))

	// 2不做任何操作，其他人过牌
	// 等待2超时，但是3不能超时，上边睡了一个50
	time.Sleep(170 * time.Millisecond)
	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfBet, 0))
	g.OnMsg(g.newPlayerActionMsg(4, abstracts.GameActionOfBet, 0))
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 3, int(g.curRound))
	assert.Equal(t, 2, int(g.curBetPlayer))

	// 等待2超时，上边睡了一个50
	time.Sleep(170 * time.Millisecond)
	// 加注，触发2超时后弃牌
	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfBet, 300))
	g.OnMsg(g.newPlayerActionMsg(4, abstracts.GameActionOfBet, 300))
	// 等待2超时
	time.Sleep(220 * time.Millisecond)
	assert.Equal(t, 4, int(g.curRound))
	assert.Equal(t, 3, int(g.curBetPlayer))
	assert.True(t, g.players[2].Discarded())

	g.OnMsg(g.newPlayerActionMsg(3, abstracts.GameActionOfBet, 0))
	g.OnMsg(g.newPlayerActionMsg(4, abstracts.GameActionOfBet, 0))
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 5, int(g.curRound))
	assert.Equal(t, 3, int(g.curBetPlayer))

	result := <- resultC
	assert.Equal(t, g.id, result.id)
	assert.Len(t, result.players, 5)

	for i, pr := range result.players {
		change, isAdd := pr.Result()
		log.L.Debug("player result", zap.Uint("player", i), zap.Uint64("result", change), zap.Bool("is add", isAdd))
	}
}

// 断言用户在某一刻的下注数量是否正确
func (g *Game) betRight(t *testing.T, player uint, shouldBe uint64) {
	assert.Equal(t, int(shouldBe), int(g.players[player].HaveBet()))
	assert.Equal(t, int(shouldBe), int(g.chipPool.playerTotalBetByRound(player)))
}

func TestGameTimer_Set(t *testing.T) {
	outC := make(chan timeoutInfo)
	timer := newGameTimer(outC)
	err := timer.Start()
	assert.NoError(t, err)
	err = timer.Start()
	assert.Error(t, err)

	timer.Set(1 * time.Millisecond, timeoutInfo{ round: 1 })
	select {
	case x := <- outC:
		assert.Equal(t, uint(1), x.round)
	}
	timer.Set(1 * time.Millisecond, timeoutInfo{ round: 1 })
	timer.Set(1 * time.Millisecond, timeoutInfo{ round: 2 })
	select {
	case x := <- outC:
		assert.Equal(t, uint(2), x.round)
	}

	err = timer.Stop()
	time.Sleep(1 * time.Millisecond)
	assert.NoError(t, err)
	err = timer.Stop()
	assert.Error(t, err)

	err = timer.Start()
	assert.NoError(t, err)
}
