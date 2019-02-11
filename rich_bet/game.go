package rich_bet

func NewGame() *Game {
	return &Game{}
}

type Game struct {

}

func (g *Game) Bet(uAddr string, amount uint64, blockHeight uint64) {

}

func (g *Game) Reward(blockHeight uint64) {

}