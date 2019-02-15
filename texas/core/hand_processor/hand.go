package hand_processor

import (
	"strconv"
	"errors"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/common/util"
)

//go:generate stringer -type=HandType
// 手牌类型（从单张开始，到皇家同花顺）
type HandType int
const (
	HandOfDZ HandType = iota
	HandOfYD
	HandOfLD
	HandOfST3
	HandOfSZ
	HandOfTH
	HandOfHL
	HandOfST4
	HandOfTHS
	HandOfHJTHS
)

func (i HandType)CnString() string {
	switch i {
	case HandOfHJTHS:
		return "皇家同花顺"
	case HandOfTHS:
		return "同花顺"
	case HandOfST4:
		return "四条"
	case HandOfHL:
		return "葫芦"
	case HandOfTH:
		return "同花"
	case HandOfSZ:
		return "顺子"
	case HandOfST3:
		return "三条"
	case HandOfLD:
		return "两对"
	case HandOfYD:
		return "一对"
	case HandOfDZ:
		return "单张"
	}
	return ""
}

// 手牌
type Hand struct {
	// 记录最原始的牌型，方便查错
	originPokers []*Poker
	pokers []*Poker
	handType HandType
	weight int
}

func (hand *Hand) GetWeight() int {
	return hand.weight
}
func (hand *Hand) GetOriginPokers() []*Poker {
	return hand.originPokers
}
func (hand *Hand) GetSortedPokers() []*Poker {
	return hand.pokers
}
func (hand *Hand) GetHandType() HandType {
	return hand.handType
}
// 牌型和权重一样大才是相等
func (hand *Hand) EqualsTo(otherH *Hand) bool {
	return hand.weight == otherH.weight && hand.handType == otherH.handType
}
//
func (hand *Hand) BiggerThan(otherH *Hand) bool {
	// 如果牌型一样才比较权重，type越大代表牌型越大
	if hand.handType == otherH.handType {
		if hand.weight > otherH.weight {
			return true
		}else {
			return false
		}
	} else if hand.handType > otherH.handType {
		return true
	} else {
		return false
	}
}

// 策略师向外暴露的接口
type IAnalyst interface {
	// 执行分析
	doAnalysis(index int, poker *Poker)
	// 最原始的牌型
	oPokers()(oPokers []*Poker)
	// 最后得到的5张最大的牌型
	cPokers()(cPokers []*Poker)
	// 保存总共有多少张扑克
	setTotalPokerCount(tCount int)
	// 获取分析后的牌型
	hType()(ht int)
	// 获取分析后的权重
	hWeight()(w int)
	// 输出分析后的手牌
	exportHand()(h *Hand)
}

// 将手牌转换成字符串手牌
func (hand *Hand)HandToHandStr() *string {
	var result string
	return &result
}

// 将字符串手牌转换成对象手牌
func HandStrToHand(handStr string) (*Hand, error) {
	handStrLen := len(handStr)
	// 必须要2的倍数
	if handStrLen % 2 != 0 {
		return nil, errors.New("手牌字符串长度必须是2的倍数")
	}
	pokersStr := []string{}
	for i := 0; i < handStrLen / 2; i++ {
		pokersStr = append(pokersStr, handStr[(i*2):(i+1)*2])
	}
	// 不足5张牌就是不合格的牌
	if len(pokersStr) < 5 {
		return nil, errors.New("手牌长度不足以计算结果")
	}
	return pokersStrToHand(pokersStr)
}

// 将字符串数组牌组转换成hand，在这里就可以直接进行牌型判断了，没必要出去了再对所有牌进行遍历
func pokersStrToHand(pokersStr []string) (*Hand, error) {
	psLen := len(pokersStr)
	// 有赖子的5张必须要用这个，因此用这个算法就行了
	analyst := DefaultAnalyst2()

	analyst.setTotalPokerCount(psLen)
	for index, pokerStr := range pokersStr{
		poker := PokerStrToPoker(pokerStr)
		if poker == nil {
			return nil, errors.New("手牌中有不合法的牌")
		}
		// 一副牌中不可能出现两张一样的牌
		for _, tmpPoker := range analyst.oPokers() {
			if poker.EqualTo(tmpPoker){
				return nil, errors.New("同一手牌中出现了两张一样的牌")
			}
		}
		// 分析牌型
		analyst.doAnalysis(index, poker)
	}
	hand := analyst.exportHand()
	return hand, nil
}

// 批量转换手牌
func HandsStrToHands(handsStr []string, resultHands map[HandType][]*Hand) {
	for _, handStr := range handsStr{
		hand, _ := HandStrToHand(handStr)
		if hand != nil {
			HandToCollection(hand, resultHands)
		}else {
			//fmt.Println("忽略错误手牌：", handStr)
		}
	}
}

// 将手牌分类装入容器中
func HandToCollection(hand *Hand, hands map[HandType][]*Hand) {
	// 这里可以考虑分类之后直接排序
	tmpHands := hands[hand.handType]
	hands[hand.handType] = append(tmpHands, hand)
}

func (hand *Hand)ToString() string {
	result := ""
	if hand == nil {
		return "nil"
	}
	result += "origin:"
	for _, poker := range hand.originPokers{
		if poker == nil {
			result += "nil"
		}else{
			result += poker.toString()
		}
	}
	result += ", checked:"
	for _, p := range hand.pokers {
		if p == nil {
			result += "nil"
		}else{
			result += p.toString()
		}
	}
	result += ", handType:" + strconv.Itoa(int(hand.handType))
	result += ", handWeight:" + strconv.Itoa(hand.weight)
	return result
}

// 将手牌组合转换成字符串
func HandsCollectionToString(hands map[int][]*Hand) string {
	result := make(map[int][]string)
	for handType, typedHands := range hands {
		for _, hand := range typedHands {
			result[handType] = append(result[handType], hand.ToString())
		}
	}
	return util.StringifyJson(result)
}

// 比较两个牌的大小，otherH大返回2，hand大返回1，相等返回0
func (hand *Hand) Match(otherH *Hand) int {
	// 如果牌型一样才比较权重，type越大代表牌型越大
	if hand.handType == otherH.handType {
		if hand.weight == otherH.weight {
			return 0
		}else if hand.weight > otherH.weight {
			return 1
		}else {
			return 2
		}
	}else if hand.handType > otherH.handType {
		return 1
	}else {
		return 2
	}
}

func (hand *Hand) HandType() int {
	return int(hand.handType)
}

func (hand *Hand) Weight() int {
	return hand.weight
}