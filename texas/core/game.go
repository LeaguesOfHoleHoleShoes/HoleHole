package core

import (
	"time"
	"errors"
	"go.uber.org/zap"
	"sort"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/texas/abstracts"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/log"
)

var (
	// 测试时会修改该值
	betTimeout = 10 * time.Second
)

type gameMsgSender interface {
	// 发送消息给单个用户
	SendMsg(playerID string, msgType int, msgID int64, msg interface{})
	// 广播消息给桌子上的所有人
	BroadcastMsg(msgType int, msgID int64, msg interface{})
}

// 初始化一个game，随后调用Run获得执行结果
func NewGame(xmBet uint64, players map[uint]abstracts.Player, sender gameMsgSender, resultChan chan *GameResult) *Game {
	g := &Game{
		id: time.Now().UnixNano(),
		xmBet: xmBet, players: players, playersLen: uint(len(players)),
		msgSender: sender,
		handMatcher: &HMatcher{},
		cardHeap: newPokerHeap(),
		msgChan: make(chan abstracts.PlayerActionMsg), timer: newGameTimer(nil),
		canLeaveChan: make(chan *canLeaveMsg),
		gameSceneChan: make(chan gameSceneMsg),
		gameStatus: gameStatus{
			chipPool: newTermChipPool(),
			// 一开始由大盲左边第一个开始下注（D用户始终为0）
			curBetPlayer: 3,
			curRound: 1,
			startBetAt: 3,
		},
		resultChan: resultChan,
	}
	return g
}

type GameResult struct {
	id int64
	players map[uint]abstracts.Player
}

// D为0，因此第一轮开始下注位置为3，后三轮开始下注位置为1
type gameStatus struct {
	chipPool     *termChipPool
	curBetPlayer uint
	curRound     uint
	// 标记从哪个位置开始bet的
	startBetAt uint
	// 用于判断游戏是否提前结束或是是否需要继续通知用户下注
	discardedPlayerCount uint
	// 记录all in的用户个数
	allInnedPlayerCount uint
	// 公共牌
	commonPokers []abstracts.Poker
}

type Game struct {
	id int64
	// 游戏的配置
	// 小盲应该下注多少，大盲是他的两倍
	xmBet uint64
	// 用户个数
	playersLen uint
	// 当前轮的所有用户。从D为0开始，顺时针一次递增1，D顺数1、2个为小盲和大盲，因此小盲是1，大盲是2
	players map[uint]abstracts.Player
	msgSender gameMsgSender

	// 游戏状态
	gameStatus
	handMatcher abstracts.HandMatcher
	cardHeap abstracts.CardHeap

	canLeaveChan chan *canLeaveMsg
	msgChan chan abstracts.PlayerActionMsg
	gameSceneChan chan gameSceneMsg
	// 工具类都用指针，只有小的纯数据类不用指针
	timer *gameTimer

	stopChan chan struct{}
	resultChan chan *GameResult
}

func (g *Game) ID() int64 {
	return g.id
}

/*

timeout后带上下一步操作？还是timeout后根据当前状态来确定下一步？因为存在下边的第二个问题，因此超时必须带上状态
两个可能要注意：
timeout处理后进来一个不合法的msg
msg处理中触发了timeout

广播消息需要是异步的，该流程不能因为某一个人接收信息过慢就塞住

超时后有两种做法，1是过，2是弃牌（必须下注时）

*/
func (g *Game) loop() {
	log.L.Debug("start game loop")
	defer func() { g.stopChan = nil }()
	for {
		select {
		case msg := <- g.msgChan:
			//log.L.Debug("on new game msg", zap.String("uid", msg.UserID))
			g.onMsg(msg)
		case info := <- g.timer.timeoutChan:
			g.onTimeout(info)
		case msg := <- g.gameSceneChan:
			g.doGetScene(msg)
		case msg := <- g.canLeaveChan:
			g.canLeave(msg)
		case <- g.stopChan:
			g.timer.Stop()
			return
		}
	}
}

// 判断用户是否可以离开，弃牌则可以离开了
func (g *Game) canLeave(msg *canLeaveMsg) {
	for _, p := range g.players {
		if p.ID() == msg.uID {
			// 弃牌则可以离开
			if p.Discarded() {
				msg.resultChan <- true
			} else {
				msg.resultChan <- false
			}
			return
		}
	}
	log.L.Warn("player call leave, but can't find him", zap.String("u id", msg.uID))
	// 找不到则说明没他，当然可以离开
	msg.resultChan <- true
}

func (g *Game) OnMsg(msg abstracts.PlayerActionMsg) {
	if g.stopChan == nil {
		log.L.Warn("game not started, but receive game msg", zap.String("uid", msg.UserID))
		return
	}
	g.msgChan <- msg
}

/*

处理客户端发送的不同消息

*/
func (g *Game) onMsg(msg abstracts.PlayerActionMsg) {
	// 下边都用g里边的变量就能保证不越界
	if g.curBetPlayer != msg.Player || g.curRound != msg.Round {
		log.L.Debug("invalid msg, cur player or cur round not match", zap.Uint("p should", g.curBetPlayer), zap.Uint("msg p", msg.Player), zap.Uint("r should", g.curRound), zap.Uint("msg round", msg.Round))
		// todo send invalid player or round
		//g.msgSender.SendMsg(g.players[msg.Player].ID(), )
		return
	}
	p := g.players[g.curBetPlayer]
	if p.AllInned() || p.Discarded() {
		log.L.Debug("can't do action", zap.String("uid", p.ID()), zap.Bool("AllInned", p.AllInned()), zap.Bool("Discarded", p.Discarded()))
		// todo send player already all in or discarded
		return
	}
	// 执行用户的操作
	switch msg.ActionType {
	case abstracts.GameActionOfBet:
		// 如果all in，在里边会标记
		enough, isAllIn := p.Bet(msg.Amount)
		if !enough {
			log.L.Debug("chip not enough", zap.String("player", p.ID()), zap.Uint64("remain", p.RemainChip()), zap.Uint64("msg.Amount", msg.Amount))
			// todo send not enough amount
			return
		}
		if isAllIn {
			g.allInnedPlayerCount++
		}
		if msg.Amount > 0 {
			log.L.Debug("player bet", zap.String("player", p.ID()), zap.Uint("round", g.curRound), zap.Uint64("amount", msg.Amount), zap.Bool("is all in", isAllIn))
			g.chipPool.bet(g.curRound, g.curBetPlayer, msg.Amount, isAllIn)
		} else {
			log.L.Debug("player bet nothing", zap.String("player", p.ID()), zap.Uint("round", g.curRound))
		}
	case abstracts.GameActionOfDiscard:
		log.L.Debug("player discard", zap.String("player", p.ID()))
		p.Discard()
		g.discardedPlayerCount++
	}
	g.afterPlayerActionOrTimeout()
}

type canLeaveMsg struct {
	uID string
	resultChan chan bool
}
func (g *Game) CanLeave(uID string) bool {
	if g.stopChan == nil {
		log.L.Warn("call can leave, but game not running", zap.String("u id", uID))
		return true
	}
	resultC := make(chan bool)
	g.canLeaveChan <- &canLeaveMsg{ uID: uID, resultChan: resultC }
	return <- resultC
}

/*

判断本轮是否结束
判断桌面是否弃牌到只有一个人持牌了
判断是否除了弃牌就是all in了
或是如果最大投注数为0且下一个投注的人是本轮开始投注的那个人，那么就进入下一轮
判断下一个应该投注的用户已投的筹码是否不为0且与最大投注数相等，是则进入下一轮

*/
func (g *Game) afterPlayerActionOrTimeout() {
	// 只有一个人没有弃牌
	if g.discardedPlayerCount + 1 == g.playersLen {
		g.allDiscardedEnd()
		return
	}

	nextBetPlayer := g.nextBetPlayer(g.curBetPlayer)
	//log.L.Debug("next player", zap.Uint("nextBetPlayer", nextBetPlayer), zap.Bool("is all in", g.players[nextBetPlayer].AllInned()))
	// 在不进入下一轮的情况下，下一个下注者就是 nextBetPlayer。因此这里设置了，如果进了setupNewRound，该值会被修正
	g.curBetPlayer = nextBetPlayer

	// 判断是否进入下一轮
	curRoundMaxAmount := g.chipPool.maxBetAmountAt(g.curRound)
	// 所有人都没投注，进入下一轮
	if curRoundMaxAmount == 0 && g.startBetAt == nextBetPlayer {
		log.L.Debug("no one bet, enter next round")
		g.setupNewRound()

	// 下一个人下的注已经是最多的了，因此本轮不需要继续下注了
	} else if curRoundMaxAmount != 0 && g.chipPool.playerHaveBetToMax(g.curRound, nextBetPlayer) {
		log.L.Debug("next player is bet max, enter next round")
		g.setupNewRound()
	}

	// 判断游戏结束
	if g.curRound > 4 {
		g.end()
		return
	}

	// todo 发出下注通知

	// 设置超时
	g.timer.Set(betTimeout, timeoutInfo{ round: g.curRound, player: g.curBetPlayer })
}

func (g *Game) dealCards() {
	// todo 发出消息通知客户端
	// round是从1开始的
	switch g.curRound {
	case 1:
		// 每人发两张牌
		for _, p := range g.players {
			p.GotPokers(g.cardHeap.DispatchPokers(2))
			// 不能广播，因为每人都只能收到自己的手牌，不能收到别人的手牌
		}
	case 2:
		// 发三张公共牌
		g.commonPokers = append(g.commonPokers, g.cardHeap.DispatchPokers(3)...)
	case 3, 4:
		// 发一张公共牌
		g.commonPokers = append(g.commonPokers, g.cardHeap.DispatchPokers(1)...)
	default:
		log.L.Warn("invalid round for dealCards", zap.Uint("cur round", g.curRound))
	}
}

// 将结果输出到玩家中，用于最终输出结果
func (g *Game) mergeResultToPlayers(r map[uint]uint64) {
	for i, pr := range  r {
		g.players[i].WinChip(pr)
	}
}

/*

弃牌结束

找到没有弃牌的人，将筹码池中所有筹码都发给该玩家

*/
func (g *Game) allDiscardedEnd() {
	log.L.Debug("game allDiscardedEnd")
	for i, p := range g.players {
		if !p.Discarded() {
			log.L.Info("all discarded end", zap.String("winner id", p.ID()), zap.Uint("player index", i))
			g.mergeResultToPlayers(g.chipPool.finalize([][]uint{ { i } }))
			break
		}
	}
	g.stop()
}

/*

所有人all in或弃牌了，直接发牌到最后

*/
func (g *Game) dealingCardsToEnd() {
	log.L.Debug("game dealingCardsToEnd")
	// 进来时round尚未++
	// 发牌到最后
	for g.curRound < 4 {
		g.curRound++
		g.dealCards()
	}
	// 随后就是正常结束的流程，end里边会调stop
	g.end()
}

/*

正常结束

比较每个人牌面大小，将结果递给筹码池处理

*/
func (g *Game) end() {
	//log.L.Debug("game end")
	g.mergeResultToPlayers(g.chipPool.finalize(g.rankPlayers()))
	g.stop()
}

func (g *Game) rankPlayers() (result [][]uint) {
	// 将相同牌的玩家放同一个数组中
	var notDiscardedPlayers [][]abstracts.Player
out:
	for _, p := range g.players {
		if !p.Discarded() {
			// 检查是否牌型相同
			for psIndex, ps := range notDiscardedPlayers {
				if g.handMatcher.Cmp(ps[0].GetHand(g.commonPokers), p.GetHand(g.commonPokers)) == 0 {
					notDiscardedPlayers[psIndex] = append(ps, p)
					continue out
				}
			}
			// 如果没有牌型相同的则新加一个
			notDiscardedPlayers = append(notDiscardedPlayers, []abstracts.Player{ p })
		}
	}

	// 对这些数组排序
	sort.Slice(notDiscardedPlayers, func(i, j int) bool {
		// 大的排前，i在后，j在前，true代表换位，因此i>j则需要返回true
		if g.handMatcher.Cmp(notDiscardedPlayers[i][0].GetHand(g.commonPokers), notDiscardedPlayers[j][0].GetHand(g.commonPokers)) == 1 {
			return true
		}
		return false
	})

	// 转成 [][]uint
	for _, ps := range notDiscardedPlayers {
		var tmp []uint
		for _, p := range ps {
			tmp = append(tmp, p.PlaceIndex())
		}
		result = append(result, tmp)
	}
	return
}

func (g *Game) stop() {
	close(g.stopChan)
}

func (g *Game) setupNewRound() {
	// 判断场上是否只有1个人能操作了，是的话则发牌到最后结束游戏
	if g.discardedPlayerCount + g.allInnedPlayerCount + 1 == g.playersLen {
		g.dealingCardsToEnd()
		return
	}

	g.curRound++
	// 在第一个下注轮中，大盲注左边的玩家第一个行动。从第二个下注轮开始，由D位置左边的第一个玩家开始行动。不能是已经弃牌和all in的玩家，否则逻辑会卡死
	sAt := g.nextBetPlayer(0)
	g.startBetAt = sAt
	g.curBetPlayer = sAt

	log.L.Debug("setup new round", zap.Uint("round", g.curRound), zap.Uint("start at", sAt))
	if g.curRound < 5 {
		// 发牌
		g.dealCards()
	}
}

func (g *Game) nextBetPlayer(cur uint) uint {
	next := g.nextPlayer(cur)
	nextP := g.players[next]
	// 找到既没有弃牌的也没有all in的
	for (nextP.AllInned() || nextP.Discarded()) && next != cur {
		next = g.nextPlayer(next)
		nextP = g.players[next]
	}
	// 可能找到了既没弃牌又没all in的人。也可能next == cur，说明接下来不需要用户操作了
	return next
}
func (g *Game) nextPlayer(cur uint) uint {
	next := cur + 1
	if next >= g.playersLen {
		return 0
	}
	return next
}

/*

处理超时

*/
func (g *Game) onTimeout(info timeoutInfo) {
	// 下边都用g里边的变量就能保证不越界
	if g.curBetPlayer != info.player || g.curRound != info.round {
		return
	}
	p := g.players[g.curBetPlayer]
	if p.AllInned() || p.Discarded() {
		return
	}
	log.L.Debug("on game Timeout", zap.Uint("round", info.round), zap.Uint("player", info.player))
	// 如果他下注等于当前最大下注值那么就是过牌，否则执行弃牌
	if !g.chipPool.playerHaveBetToMax(g.curRound, g.curBetPlayer) {
		log.L.Debug("timeout discard", zap.Uint("round", info.round), zap.Uint("player", info.player))
		p.Discard()
		g.discardedPlayerCount++
	}
	g.afterPlayerActionOrTimeout()
}

/*

执行游戏逻辑，返回执行结果
每局都新建game，因此不存在多协程同时调用Run的情况

*/
func (g *Game) Run() {
	if g.stopChan != nil {
		panic("game already started")
	}
	g.stopChan = make(chan struct{})
	g.doStart()
	// 阻塞至loop stop，则game结束，返回结果
	g.loop()
	// send result
	g.resultChan <- &GameResult{
		id: g.id,
		players: g.players,
	}
	return
}

// 在loop之前执行开始操作
func (g *Game) doStart() {
	g.dealCards()
	// 下大小盲，广播当前下注的玩家
	g.players[1].Bet(g.xmBet)
	g.chipPool.bet(1, 1, g.xmBet, false)
	g.players[2].Bet(g.xmBet * 2)
	g.chipPool.bet(1, 2, g.xmBet * 2, false)

	// 启动timer
	g.timer.Start()
	g.timer.Set(betTimeout, timeoutInfo{ round: g.curRound, player: g.curBetPlayer })
}

/*

整局游戏的筹码池
并在最后结算筹码分配

*/

func newTermChipPool() *termChipPool {
	return &termChipPool{
		roundMaxAmount: map[uint]uint64{},
		roundTotalBet: map[uint]map[uint]uint64{},
		pool: newChipPool(1),
	}
}

type termChipPool struct {
	// 记录某一轮最大下注数量 K round V amount
	roundMaxAmount map[uint]uint64
	// 记录某轮某个用户下注数量 K round V （K player V amount）
	roundTotalBet map[uint]map[uint]uint64

	pool *chipPool
}

/*

传入没有弃牌的人的排名，0号位为第一名，以此类推，同一名次可能有多个玩家
返回每个人获得桌面的筹码个数，输家（赢得0个）的人不在结果集中

*/
func (p *termChipPool) finalize(winners [][]uint) map[uint]uint64 {
	result := map[uint]uint64{}
	nextPool := p.pool
	for nextPool != nil {
		tmpR := nextPool.finalize(winners)
		// 汇总结果
		for u, r := range tmpR {
			result[u] += r
		}
		nextPool = nextPool.nextPool
	}
	return result
}

/*

执行用户下注
首先判断是否要分池
前置分池：在有人all in之前检查到桌面上有人筹码不够则首先分池（弃，第一筹码少的人不一定再继续下注，第二这里不方便去遍历桌上所有人的筹码）
后置分池：在有人all in时检查是否够池子标准，不够则分池。这里采用后置分池更合理

*/
func (p *termChipPool) bet(round uint, player uint, amount uint64, isAllIn bool) error {
	rt := p.getRoundTotalBet(round)
	curURoundTotal := rt[player] + amount
	if !isAllIn && curURoundTotal < p.maxBetAmountAt(round) {
		return errors.New("bet not enough")
	}
	// 设置本轮下注的最大值
	if curURoundTotal > p.roundMaxAmount[round] {
		p.roundMaxAmount[round] = curURoundTotal
	}
	rt[player] = curURoundTotal
	// 向筹码池中下注，如果在某个筹码池则需要分池
	nextPool := p.pool
	remainAmount := amount
	loopCount := 0
	for remainAmount > 0 {
		loopCount++
		if loopCount > 20 {
			panic("infinity loop")
		}
		// 往pool中下注，重新赋值remainAmount，需要在里边做分池。如果是最后一个池子，那么amount必须在里边分配完
		remainAmount = nextPool.bet(round, player, remainAmount, isAllIn, nextPool.nextPool == nil)
		nextPool = nextPool.nextPool
	}
	return nil
}

func (p *termChipPool) maxBetAmountAt(round uint) uint64 {
	return p.roundMaxAmount[round]
}

func (p *termChipPool) playerHaveBetAt(round uint, player uint) uint64 {
	total := p.getRoundTotalBet(round)
	return total[player]
}

func (p *termChipPool) playerHaveBetToMax(round uint, nextP uint) bool {
	total := p.getRoundTotalBet(round)
	return total[nextP] == p.roundMaxAmount[round]
}

func (p *termChipPool) getRoundTotalBet(round uint) map[uint]uint64 {
	tmp := p.roundTotalBet[round]
	if tmp != nil {
		return tmp
	}
	tmp = map[uint]uint64{}
	p.roundTotalBet[round] = tmp
	return tmp
}

func (p *termChipPool) playerTotalBetByChildPool(player uint) uint64 {
	return p.pool.playerTotalBet(player)
}

func (p *termChipPool) playerTotalBetByRound(player uint) (total uint64) {
	for _, rt := range p.roundTotalBet {
		total += rt[player]
	}
	return
}

/*

可能出现有人下注比别人多的情况（只有最后两人有该可能，多余两人则应该服从分池逻辑）
因此如果出现均分情况时，应该先将多余的筹码返给多下的人，再做均分操作

*/

func newChipPool(round uint) *chipPool {
	return &chipPool{ round: round, total: map[uint]uint64{} }
}

// todo close池子的时机，在一轮结束后，所有池子
type chipPool struct {
	// 记录其属于哪个round
	round uint
	// 记录当前pool的单个最大下注值
	maxAmount uint64
	// 标记该pool是否有人all in
	haveAllIn bool
	// 记录该池每个人一共下了多少筹码
	total map[uint]uint64
	// 下一个pool
	nextPool *chipPool
}

// 计算该pool及其下所有pool的筹码总量
func (p *chipPool) playerTotalBet(player uint) uint64 {
	total := p.total[player]
	for next := p.nextPool; next != nil; next = next.nextPool {
		total += next.total[player]
	}
	return total
}

func (p *chipPool) totalChip() (result uint64) {
	for _, pt := range p.total {
		result += pt
	}
	return
}

// 如果用户下注比当前池子少则需要分池，人多的池子放在前
// 如果下注的人多余桌面，而桌面中又有人all in过
func (p *chipPool) bet(round uint, player uint, amount uint64, isAllIn bool, isLastPool bool) (remainAmount uint64) {
	curUTotal := p.total[player] + amount
	//fmt.Println("bet", player, curUTotal, isAllIn, isLastPool)
	//fmt.Println(p.total, p.maxAmount)
	// 在有人all时，当前人下注大于池子最大数
	if isLastPool && p.haveAllIn && curUTotal > p.maxAmount {
		//fmt.Println("splitByMore", player)
		p.splitByMore(round, player, curUTotal, isAllIn)
		return 0
	}
	// 要在上个if后边，排除自己all in
	if isAllIn {
		p.haveAllIn = true
	}
	// 不够当前池子，要分池
	if curUTotal < p.maxAmount {
		// split里边设置了max amount
		p.splitByLess(round, player, curUTotal)
		return 0
	}
	// 如果是last pool则所有多余的筹码都放在这个Pool中
	if isLastPool {
		p.maxAmount = curUTotal
		p.total[player] = curUTotal
		return 0
	}
	// 在下多注的人在所有该下注人的中间时可能出现这种情况，也就是分池后，还会往低池中放入筹码
	// 不是last pool，则说明已经分池，当前pool max amount不能变化，只能是用户少筹码补进来
	// p.maxAmount - p.total[player]为当前用户还需往池子中下多少注，amount减去该值后就是剩余的筹码
	remainAmount = amount - (p.maxAmount - p.total[player])
	p.total[player] = p.maxAmount
	return
}

/*

执行分池

新生成一个池子放入别人多余的筹码进去，多人的筹码池放在前

*/

/*
当前下注人all in比池子中的多，并且池子中有人all过了
这里将all in标记传进来是因为该用户多余的筹码会放到下一个筹码池中，如果他是all in状态，那么该状态也应该传递到下一个池子中。而splitByLess不用带入该状态，因为那种情况下注用户不可能进入下一个池子中。
*/
func (p *chipPool) splitByMore(round uint, player uint, curUTotal uint64, isAllIn bool) {
	newPool := newChipPool(round)
	newPool.haveAllIn = isAllIn
	newPool.nextPool = p.nextPool
	p.nextPool = newPool

	// 设置用户在这两个池子中的筹码数
	p.total[player] = p.maxAmount
	remain := curUTotal - p.maxAmount
	newPool.total[player] = remain
	newPool.maxAmount = remain
}

// 当前下注人all in比池子中的少
func (p *chipPool) splitByLess(round uint, player uint, curUTotal uint64) {
	newPool := newChipPool(round)
	newPool.nextPool = p.nextPool
	p.nextPool = newPool

	p.maxAmount = curUTotal
	// player可能不在map中，因此需要在外边赋值
	p.total[player] = curUTotal
	for k, v := range p.total {
		// 当前下注人只用纳入本池，不用纳入下一个筹码池
		if k == player {
			continue
		}
		// 少于或等于该筹码的不需要纳入下一个筹码池
		if v <= curUTotal {
			continue
		}
		// 其他人多余的下注放入下一个筹码池，并且其在该池的筹码变成curUTotal
		dis := v - curUTotal
		newPool.total[k] = dis
		// 设置下个池子正确的max
		if dis > newPool.maxAmount {
			newPool.maxAmount = dis
		}
		p.total[k] = curUTotal
	}
}

/*

传入没有弃牌的人的排名，0号位为第一名，一次类推，同一名次可能有多个玩家
返回每个人获得桌面的筹码个数，输家（赢得0个）的人不在结果集中

是否存在弃牌退款的情况分析（实际不存在）：
1. 在上层执行多退操作，只有最后一个pool可能存在某人筹码下多了的情况，多余的钱会被放在新的池子里，而该池子至少会有一个人不弃牌
2. 会不会出现下注少的人没弃牌，但是下注多的人都弃牌了的情况？不会出现这种情况，最后始终会剩一个下注最多那级的人，随后触发所有人都all in或弃牌，发牌到最后然后自动结算。因此不会出现弃牌退款的情况，因为始终会有最后一个多注的人触发发牌到最后

*/
func (p *chipPool) finalize(winners [][]uint) map[uint]uint64 {
	result := map[uint]uint64{}
	// 从排名开始往下发放奖励
	for _, ws := range winners {
		matchedWs := p.matchWinners(ws)
		mLen := len(matchedWs)
		if mLen > 0 {
			// 余数是抽成
			avg := p.allPlayerTotalChip() / uint64(mLen)
			for _, w := range matchedWs {
				result[w] = avg
			}
			break
		}
	}
	return result
}

func (p *chipPool) allPlayerTotalChip() (result uint64) {
	for _, c := range p.total {
		result += c
	}
	return
}

func (p *chipPool) matchWinners(ws []uint) (result []uint) {
	// 检查该等级的赢家中是否在本pool中下注，有则加入result中
	for _, w := range ws {
		for p := range p.total {
			if w == p {
				result = append(result, w)
				break
			}
		}
	}
	return
}

/*

1. 没超时就被重置
2. 超时像外部发送通知

测试点
1. 没超时重置后依旧可以用
2. 超时后外部会收到超时通知，并且重置后还可以用

*/

func newGameTimer(toutChan chan timeoutInfo) *gameTimer {
	// game 会直接用timeoutChan变量，因此可以传nil进来
	if toutChan == nil {
		toutChan = make(chan timeoutInfo)
	}
	return &gameTimer{ timeoutChan: toutChan }
}

type timeoutInfo struct {
	round uint
	player uint
}

type gameTimer struct {
	t *time.Timer
	info timeoutInfo

	timeoutChan chan timeoutInfo
	stopChan chan struct{}
}

func (t *gameTimer) Start() error {
	if t.stopChan != nil {
		return errors.New("timer already started")
	}
	t.stopChan = make(chan struct{})
	t.t = time.NewTimer(0)
	t.t.Stop()
	go t.loop()
	return nil
}

func (t *gameTimer) loop() {
	for {
		select {
		case <- t.t.C:
			t.timeoutChan <- t.info

		case <- t.stopChan:
			t.t.Stop()
			t.t = nil
			log.L.Debug("gameTimer loop finished")
			return
		}
	}
}

func (t *gameTimer) Set(d time.Duration, info timeoutInfo) {
	t.t.Reset(d)
	t.info = info
}

func (t *gameTimer) Stop() error {
	if t.stopChan == nil {
		return errors.New("timer already stopped")
	}
	close(t.stopChan)
	t.stopChan = nil
	return nil
}