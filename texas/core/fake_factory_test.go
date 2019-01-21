package core

import (
	"strconv"
	"time"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/texas/abstracts"
)

type fakeMsgSender struct {}

func (s *fakeMsgSender) SendMsg(playerID string, msgType int, msgID int64, msg interface{}) {}

func (s *fakeMsgSender) BroadcastMsg(msgType int, msgID int64, msg interface{}) {}

type fakeUser struct {
	uid string
	balance uint64
}

func (u *fakeUser) ID() string { return u.uid }

func (u *fakeUser) Copy() abstracts.User { return &fakeUser{ uid: u.uid } }

func (u *fakeUser) Balance() uint64 { return u.balance }

func (u *fakeUser) ChangeBalance(dis uint64, isAdd bool) {
	if isAdd {
		u.balance += dis
	} else {
		u.balance -= dis
	}
}

func newPlayerWithFakeUser(pIndex uint, maxBringIn uint64) *Player {
	u := &fakeUser{ uid: strconv.Itoa(int(pIndex)), balance: 10000 }
	return NewPlayer(pIndex, u, maxBringIn)
}

func newFakePlayersInTable1() map[uint]abstracts.Player {
	return map[uint]abstracts.Player{
		0: newPlayerWithFakeUser(0, 2000),
		1: newPlayerWithFakeUser(1, 2000),
		2: newPlayerWithFakeUser(2, 2000),
		3: newPlayerWithFakeUser(3, 2000),
		4: newPlayerWithFakeUser(4, 2000),
	}
}

func newFakePlayersInTable2() map[uint]abstracts.Player {
	return map[uint]abstracts.Player{
		0: newPlayerWithFakeUser(0, 2000),
		1: newPlayerWithFakeUser(1, 2000),
		2: newPlayerWithFakeUser(2, 1500),
		3: newPlayerWithFakeUser(3, 2000),
		4: newPlayerWithFakeUser(4, 2000),
	}
}

func (g *Game) newPlayerActionMsg(player uint, action abstracts.GameAction, amount uint64) abstracts.PlayerActionMsg {
	return abstracts.PlayerActionMsg{
		// 不能是客户端传上来的，应该有程序赋值
		MsgID: time.Now().UnixNano(),
		UserID: strconv.Itoa(int(player)),
		Player: player,

		GameID: g.id,
		Round: g.curRound,
		ActionType: action,
		// 下注情况下是下多少注。如果是过牌该值就为0
		Amount: amount,
	}
}
