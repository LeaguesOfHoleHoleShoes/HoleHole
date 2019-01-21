package abstracts

type GameAction uint

const (
	GameActionOfBet     GameAction = iota
	GameActionOfDiscard
)

const (
	// c - s
	MsgTypeQuickStart = 0x10
	// c - s
	MsgTypeLeave = 0x11
	// s - c
	MsgTypePrepare = 0x12
	// s - c
	MsgTypeNotReadyLeave = 0x13
	// s - c
	MsgTypeNotEnoughBalanceLeave = 0x14
	// c - s
	MsgTypeReady = 0x15
	// c - s
	MsgTypeGameAction = 0x16

	// s - c
	MsgTypeErr = 0x20
	// s - c
	MsgTypeSuccess = 0x21
	// s - c
	MsgTypeTableScene = 0x22
)

type CommonMsg struct {
	MsgID int64
	User User
}

type PlayerActionMsg struct {
	// 不能是客户端传上来的，应该有程序赋值
	MsgID int64
	UserID string
	Player uint

	GameID     int64 `json:"game_id"`
	Round      uint `json:"round"`
	ActionType GameAction `json:"action_type"`
	// 下注情况下是下多少注。如果是过牌该值就为0
	Amount uint64 `json:"amount"`
}

type ErrResp struct {
	ErrCode int `json:"err_code"`
	Info string `json:"info"`
}

type SuccessResp struct {
	Info string `json:"info"`
}

/*

当前桌子的快照
1. 椅子情况：玩家信息，玩家剩余筹码数，自己的手牌，当前D，当前该谁出牌，每个位置是弃牌、all in、正常状态
1. 公共牌
1. 筹码池

*/
type TableScene struct {
	CurD int `json:"cur_d"`
	CurBet int `json:"cur_bet"`
	Players []*PlayerScene `json:"players"`
	CommonPokers []*PokerScene `json:"common_pokers"`
	ChipPools []*ChipPoolScene `json:"chip_pool"`
}


/*
1. 椅子情况：玩家信息，玩家剩余筹码数，自己的手牌，当前D，当前该谁出牌，每个位置是弃牌、all in、正常状态
1. 公共牌
1. 筹码池
*/
type GameScene struct {
	// uid
	CurBet       string
	// uid
	Players      map[string]*PlayerScene
	CommonPokers []*PokerScene
	ChipPools    []*ChipPoolScene
}

const (
	PlayerStatusNormal = iota
	PlayerStatusDiscarded
	PlayerStatusAllInned
)

type PlayerScene struct {
	UserID string `json:"user_id"`
	RemainChip uint64 `json:"remain_chip"`
	Pokers []*PokerScene `json:"pokers"`
	// 当前状态，弃牌、all in、正常
	Status int `json:"status"`
}

type PokerScene struct {
	Whole string `json:"whole"`
}

type ChipPoolScene struct {
	Chips uint64 `json:"chips"`
}
