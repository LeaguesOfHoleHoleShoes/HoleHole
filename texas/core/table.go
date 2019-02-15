package core

import (
	"errors"
	"fmt"
	"time"
	"go.uber.org/zap"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/texas/abstracts"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/common/log"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/common/util"
)

type msgSender interface {
	Send(id string, msgType int, mID int64, msg []byte)
}

func NewTable(id int, seatCount int, level TableLevel, msgSender msgSender) *Table {
	timer := time.NewTimer(time.Second)
	timer.Stop()
	return &Table{
		id: id, level: level, seats: make([]abstracts.User, seatCount),
		seatCount: seatCount, msgSender: msgSender,
		prepareStartTimer: timer,
		getSceneChan: make(chan getSceneMsg, 1),
		readyChan: make(chan withErrMsg, 1),
		enterChan: make(chan withErrMsg, 1),
		leaveChan: make(chan withErrMsg, 1),
		actionChan: make(chan actionMsg, 1),
		gameFinishedChan: make(chan *GameResult, 1),
	}
}

/*

用户加入桌子
用户坐下
检查是否要开始游戏
开始游戏后执行游戏逻辑并得到执行结果
提交结果后再回到检查是否要开始游戏

要记录当前桌子的状态，切换D等操作

与game中的players做映射
// 当前轮的所有用户。从D为0开始，顺时针一次递增1
players map[uint]abstracts.Player

*/
type Table struct {
	id int
	level TableLevel
	seats []abstracts.User
	seatCount int
	msgSender msgSender

	// 记录最近一次准备开始时，准备好的用户。每次准备计时结束后，都要清空该数据
	preparedUsers map[string]int
	// 要对用户回馈的prepare msg id做check
	latestPrepareMsgID int64

	// 开局后该值顺位往下第一个用户为D，
	curD int
	curGame abstracts.Game
	// 记录本局离开的用户，广播消息时过滤它，在get scene时也可以标记，开局重置该变量，结束时置为nil。key为seat index
	leavedUsers map[int]abstracts.User

	prepareStartTimer *time.Timer
	getSceneChan chan getSceneMsg
	readyChan chan withErrMsg
	enterChan chan withErrMsg
	leaveChan chan withErrMsg
	actionChan chan actionMsg
	gameFinishedChan chan *GameResult
	stopChan chan struct{}
}

func (t *Table) loop() {
	for {
		select {
		case msg := <- t.getSceneChan:
			t.doGetScene(msg)
		case <- t.prepareStartTimer.C:
			t.startGameCheck()
		case msg := <- t.readyChan:
			t.doReady(msg)
		case msg := <- t.enterChan:
			t.doEnter(msg)
		case msg := <- t.leaveChan:
			t.doLeave(msg)
		case msg := <- t.actionChan:
			t.doActionChan(msg)
		case result := <- t.gameFinishedChan:
			t.doGameFinished(result)
		case <- t.stopChan:
			return
		}
	}
}

// 检查是否可以开始游戏
func (t *Table) startGameCheck() {
	readyUser := 0
	// 检查用户是否已提交准备，如果没有则将其移除该桌子
	for i := 0; i < t.seatCount; i++ {
		u := t.seats[i]
		if u == nil {
			continue
		}

		// 检查用户筹码是否足够，不够则踢出桌子
		if u.Balance() < t.level.MinHave {
			t.msgSender.Send(u.ID(), abstracts.MsgTypeNotEnoughBalanceLeave, time.Now().UnixNano(), nil)
			// 移除该用户
			t.seats[i] = nil
		}

		// 用户没有准备，可能是已经退出或是掉线了，移除该用户
		if !t.userInReadyMap(u) {
			t.msgSender.Send(u.ID(), abstracts.MsgTypeNotReadyLeave, time.Now().UnixNano(), nil)
			// 移除该用户
			t.seats[i] = nil
		} else {
			readyUser++
		}
	}
	t.preparedUsers = nil

	if readyUser > 1 {
		t.startGame()
	}
}

func (t *Table) userInReadyMap(u abstracts.User) bool {
	for uID, exist := range t.preparedUsers {
		if uID == u.ID() && exist == 1 {
			return true
		}
	}
	return false
}

/*

做好game的映射关系，初始化game，用户带入筹码，并开始游戏

*/
func (t *Table) startGame() {
	if t.curGame != nil {
		panic("already have a game")
	}

	t.leavedUsers = map[int]abstracts.User{}
	t.curGame = NewGame(t.level.Xm, t.getPlayersFromSeats(), t, t.gameFinishedChan)
	go t.curGame.Run()
}

/*

seats转players，从D开始，获取有人的位置映射到map中，并带入筹码

*/
func (t *Table) getPlayersFromSeats() map[uint]abstracts.Player {
	result := map[uint]abstracts.Player{}
	// find cur d
	dIndex, dUser := t.nextUser(t.curD)
	t.curD = dIndex
	result[0] = NewPlayer(0, dUser, t.level.BringIn)
	log.L.Info("find cur game d", zap.Int("cur d", t.curD), zap.String("cur d id", dUser.ID()))

	i := -1
	var u abstracts.User = nil
	playerIndex := uint(1)
	count := 0
	for {
		i, u = t.nextUser(dIndex)
		if i == dIndex {
			break
		}
		result[playerIndex] = NewPlayer(playerIndex, u, t.level.BringIn)
		playerIndex++

		count++
		if count > t.seatCount {
			panic("infinite for")
		}
	}

	return result
}

// 获取下一个座位
func (t *Table) nextSeat(cur int) int {
	next := cur + 1
	if next >= t.seatCount {
		next = 0
	}
	return next
}

// 获取下一个有用户的座位
func (t *Table) nextUser(cur int) (int, abstracts.User) {
	count := 0
	index := t.nextSeat(cur)
	var user abstracts.User = nil
	for index != cur {
		if user = t.seats[index]; user != nil {
			return index, user
		}
		index = t.nextSeat(index)

		count++
		if count > t.seatCount {
			panic("infinite for")
		}
	}
	return -1, nil
}

func (t *Table) doGameFinished(result *GameResult) {
	if t.curGame == nil {
		panic("table do finish game, but game is nil")
	}
	if t.curGame.ID() != result.id {
		panic(fmt.Sprintf("table do finish game, but game id not right. cur game id: %v, result id: %v", t.curGame.ID(), result.id))
	}

	// 处理结果
	for _, p := range result.players {
		u := t.getUserByIDFromSeat(p.ID())
		if u == nil {
			log.L.Error("can't find user from seat", zap.String("uid", p.ID()))
			continue
		}
		u.ChangeBalance(p.Result())
		// todo 记录变化
	}

	// 移除离开的用户
	for seatIndex := range t.leavedUsers {
		t.seats[seatIndex] = nil
	}
	t.leavedUsers = nil
	t.curGame = nil
}

func (t *Table) getUserByIDFromSeat(id string) abstracts.User {
	for _, u := range t.seats {
		if u.ID() == id {
			return u
		}
	}
	return nil
}

/*

1. 椅子情况：玩家信息，玩家剩余筹码数，自己的手牌，当前D，当前该谁出牌，每个位置是弃牌、all in、正常状态
1. 公共牌
1. 筹码池
从game中拿到当前情况，随后转换成桌子的scene

*/
func (t *Table) doGetScene(msg getSceneMsg) {
	// 组装每个seat的状态
	gameScene := t.curGame.GetScene(msg.uID)
	result := abstracts.TableScene{
		CurD: t.curD,
		Players: make([]*abstracts.PlayerScene, t.seatCount),
		CommonPokers: gameScene.CommonPokers,
		ChipPools: gameScene.ChipPools,
	}

	for i, u := range t.seats {
		if u.ID() == gameScene.CurBet {
			result.CurBet = i
		}
		result.Players[i] = gameScene.Players[u.ID()]
	}

	msg.resultChan <- result
}

func (t *Table) doReady(msg withErrMsg) {
	if t.preparedUsers == nil {
		msg.resultChan <- errors.New("game not prepare to start")
	} else {
		t.preparedUsers[msg.user.ID()] = 1
		msg.resultChan <- nil
	}
}

// 处理用户进入桌子
// 判断是否要准备开始
func (t *Table) doEnter(msg withErrMsg) {
	sitUser := 0
	sit := false
	for i := 0; i < t.seatCount; i++ {
		if t.seats[i] == nil && !sit {
			sitUser++
			sit = true
			t.seats[i] = msg.user.Copy()
		} else if t.seats[i] != nil {
			sitUser++
		}
	}

	if sit {
		msg.resultChan <- nil
	} else {
		msg.resultChan <- errors.New("no more seat")
	}

	if sitUser > 1 {
		// 准备开始游戏
		t.latestPrepareMsgID = time.Now().UnixNano()
		t.BroadcastMsg(abstracts.MsgTypePrepare, t.latestPrepareMsgID, nil)
		t.prepareStartTimer.Reset(2 * time.Second)
		t.preparedUsers = map[string]int{}
	}
}

// 找到user，
func (t *Table) doLeave(msg withErrMsg) {
	if t.curGame == nil {
		msg.resultChan <- errors.New("game not started")
		return
	}
	if !t.curGame.CanLeave(msg.user.ID()) {
		msg.resultChan <- errors.New("you didn't discard, can't leave")
		return
	}

	for index, u := range t.seats {
		if u.ID() == msg.user.ID() {
			t.leavedUsers[index] = u
		}
	}

	msg.resultChan <- nil
}

func (t *Table) doActionChan(msg actionMsg) {
	if t.curGame == nil {
		msg.resultChan <- errors.New("game not started")
		return
	}
	t.curGame.OnMsg(msg.action)
	msg.resultChan <- nil
}

type actionMsg struct {
	action abstracts.PlayerActionMsg
	resultChan chan error
}

type withErrMsg struct {
	user abstracts.User
	resultChan chan error
}

type getSceneMsg struct {
	uID string
	resultChan chan abstracts.TableScene
}

func (t *Table) GetScene(uID string) abstracts.TableScene {
	result := make(chan abstracts.TableScene)
	t.getSceneChan <- getSceneMsg{ uID: uID, resultChan: result }
	// 如果table stop，程序必须结束，否则就可能有协程泄漏
	return <- result
}

func (t *Table) Ready(u abstracts.User) error {
	result := make(chan error)
	t.readyChan <- withErrMsg{ user: u, resultChan: result }
	return <- result
}

func (t *Table) Enter(u abstracts.User) error {
	result := make(chan error)
	t.enterChan <- withErrMsg{ user: u, resultChan: result }
	return <- result
}

func (t *Table) Leave(u abstracts.User) error {
	result := make(chan error)
	t.leaveChan <- withErrMsg{ user: u, resultChan: result }
	return <- result
}

func (t *Table) Do(action abstracts.PlayerActionMsg) error {
	result := make(chan error)
	t.actionChan <- actionMsg{ action: action, resultChan: result }
	return <- result
}

func (t *Table) SendMsg(playerID string, msgType int, mID int64, msg interface{}) {
	t.msgSender.Send(playerID, msgType, mID, util.StringifyJsonToBytes(msg))
}

func (t *Table) BroadcastMsg(msgType int, msgID int64, msg interface{}) {
	for i := 0; i < t.seatCount; i++ {
		// 不给离开的用户广播消息
		if t.leavedUsers != nil && t.leavedUsers[i] != nil {
			log.L.Debug("don't broadcast msg to the user leaved", zap.String("uid", t.leavedUsers[i].ID()))
			continue
		}
		u := t.seats[i]
		if u != nil {
			t.SendMsg(u.ID(), msgType, msgID, msg)
		}
	}
}

func (t *Table) Start() error {
	if t.stopChan != nil {
		return errors.New("already started")
	}
	t.stopChan = make(chan struct{})
	go t.loop()

	return nil
}

func (t *Table) Stop() error {
	if t.stopChan == nil {
		return errors.New("not started")
	}
	close(t.stopChan)
	t.stopChan = nil

	return nil
}

