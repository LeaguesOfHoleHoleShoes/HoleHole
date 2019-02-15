package core

import (
	"go.uber.org/zap"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/texas/abstracts"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/common/log"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/texas/core/hand_processor"
)

func NewPlayer(pIndex uint, u abstracts.User, maxBringIn uint64) *Player {
	bringIn := maxBringIn
	if u.Balance() < maxBringIn {
		bringIn = u.Balance()
	}
	return &Player{ playerIndex: pIndex, id: u.ID(), bringIn: bringIn, remain: bringIn, hand: nil }
}

type Player struct {
	playerIndex uint
	id string
	bringIn uint64
	// 下注后剩余的筹码
	remain uint64
	// 赢得的筹码
	win uint64

	discarded bool
	allInned bool

	// 玩家的手牌
	pokers []abstracts.Poker
	// 手牌+公共牌
	hand abstracts.Hand
}

// 获取比赛结果
func (p *Player) Result() (change uint64, isAdd bool) {
	total := p.remain + p.win
	// 总数比带入的多
	if total > p.bringIn {
		return total - p.bringIn, true
	} else {
		return p.bringIn - total, false
	}
}

func (p *Player) ID() string {
	return p.id
}

func (p *Player) PlaceIndex() uint {
	return p.playerIndex
}

func (p *Player) Discard() {
	p.discarded = true
}

func (p *Player) Discarded() bool {
	return p.discarded
}

func (p *Player) AllInned() bool {
	return p.allInned
}

func (p *Player) Bet(amount uint64) (enough bool, isAllIn bool) {
	if amount > p.remain {
		return false, false
	}
	enough = true
	if amount == p.remain {
		log.L.Debug("tag player all in", zap.String("player", p.id))
		isAllIn = true
		p.allInned = true
	}
	p.remain -= amount
	return
}

func (p *Player) HaveBet() uint64 {
	return p.bringIn - p.remain
}

func (p *Player) RemainChip() uint64 {
	return p.remain
}

func (p *Player) OriginChip() uint64 {
	return p.bringIn
}

func (p *Player) WinChip(amount uint64) {
	p.win += amount
}

func (p *Player) GotPokers(ps []abstracts.Poker) {
	p.pokers = append(p.pokers, ps...)
}

func (p *Player) Pokers() []abstracts.Poker  {
	return p.pokers
}

func (p *Player) GetHand(commonPokers []abstracts.Poker) abstracts.Hand {
	if p.hand != nil {
		return p.hand
	}
	handStr := ""
	for _, poker := range p.pokers {
		handStr += poker.GetWhole()
	}
	for _, poker := range commonPokers {
		handStr += poker.GetWhole()
	}
	log.L.Debug("player hand", zap.String("hand", handStr), zap.String("player id", p.id))

	var err error
	if p.hand, err = hand_processor.HandStrToHand(handStr); err != nil {
		panic("parse hand str failed: " + err.Error())
	}
	return p.hand
}
