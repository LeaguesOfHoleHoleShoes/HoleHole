package abstracts

type User interface {
	// user的id
	ID() string
	Copy() User
	// 在数据库中统一结算，带入桌子的钱不在DB里减，只有结果出来后在DB结算。可能要做此缓存，外边会经常调用该函数
	Balance() uint64
	// 处理结果
	ChangeBalance(dis uint64, isAdd bool)
}

// for game
type Player interface {
	// player的id与user匹配
	ID() string
	// 该用户的位置
	PlaceIndex() uint
	// 弃牌
	Discard()
	// 判断是否弃牌了
	Discarded() bool
	// 判断是否已经all in了
	AllInned() bool
	// 下注，返回用户是否有足够的筹码，以及是否是all in
	Bet(amount uint64) (enough bool, isAllIn bool)
	// 获取用户已经下注了多少
	HaveBet() uint64
	// 返回当前用户还剩多少筹码
	RemainChip() uint64
	// 开始这局游戏时该用户有多少筹码
	OriginChip() uint64
	// 计算结果
	Result() (change uint64, isAdd bool)
	// 赢得筹码
	WinChip(amount uint64)
	// 收到发牌
	GotPokers(ps []Poker)

	Pokers() []Poker
	// 获取结果Hand
	GetHand(commonPokers []Poker) Hand
}

type Table interface {
	Start() error
	Stop() error

	// 观看功能是没啥用的，因此一进来就让他自动带入筹码并坐下即可，每次筹码不够就自动加，直到无码可加或用户主动退出或用户掉线
	Enter(u User) error
	Leave(u User) error
	// 每次客户端程序自动发该消息，如果没有发则默认其掉线，将其踢出该桌子
	Ready(u User) error

	Do(action PlayerActionMsg) error

	GetScene(uID string) TableScene
	//TakeASeat()
	//StandUp()
}

type Game interface {
	ID() int64
	Run()
	OnMsg(msg PlayerActionMsg)
	CanLeave(uID string) bool
	GetScene(uid string) *GameScene
}

type HandMatcher interface {
	// 对比两个player的牌型大小
	// h1 > h2 return 1, h1 < h2 return -1, h1 == h2 return 0
	Cmp(h1, h2 Hand) int
}

type Hand interface {
	HandType() int
	Weight() int
}

type Poker interface {
	// face + color
	GetWhole() string
}

type CardHeap interface {
	DispatchPokers(count int) []Poker
}