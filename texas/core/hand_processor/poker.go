package hand_processor

import (
	"strings"
	"go.uber.org/zap"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/common/log"
)

//用于做扑克合理性判断
const facesStr = "23456789TJQKA"
const colorsStr = "shdc"
const laiZiStr = "z"

// poker最好不要有运行时状态
type Poker struct {
	face string
	color string
	// 存整张牌
	whole string
	// 标志这张牌是不是赖子替的牌
	bornByLz bool
}

func (poker *Poker) GetWhole() string {
	return poker.whole
}

func newLz() *Poker {
	p := new(Poker)
	p.face = laiZiStr
	p.color = laiZiStr
	p.whole = laiZiStr + laiZiStr
	p.bornByLz = false
	return p
}

// 字符串转poker
func PokerStrToPoker(str string) *Poker {
	tmpFace := str[0:1]
	tmpColor := str[1:2]
	var poker *Poker
	validPoker := false
	if strings.Contains(facesStr, tmpFace) && strings.Contains(colorsStr, tmpColor) {
		validPoker = true
	} else if tmpFace == "X" && tmpColor == "n" {
		tmpFace = laiZiStr
		tmpColor = laiZiStr
		str = laiZiStr + laiZiStr
		validPoker = true
	}
	if validPoker {
		poker = new(Poker)
		poker.face = tmpFace
		poker.color = tmpColor
		poker.whole = str
		poker.bornByLz = false
	}else {
		log.L.Error("有牌错误:", zap.String("hand", str))
	}
	return poker
}

// 判断两张牌是否一样
func (poker *Poker)EqualTo(otherP *Poker) bool {
	if poker.whole == otherP.whole {
		return true
	}else {
		return false
	}
}

// 返回下一张牌的大小是啥，用于判断是否是顺子
func (poker *Poker)nextFace() string {
	index := strings.Index(facesStr, poker.face)
	return string(facesStr[index + 1])
}

// 转成string
func (poker *Poker)toString() string {
	return poker.whole
}

// 判断判断当前的牌是否比传进来的那张更大(相等也返回false)
func (poker *Poker)biggerOrEqual(otherP *Poker) bool {
	if strings.Index(facesStr, poker.face) >= strings.Index(facesStr, otherP.face){
		return true
	} else {
		return false
	}
}

// 判断第一张牌是否是小牌
func isFirstFaceSmall(first string, second string) bool {
	//if first == "" || second == "" {
	//	panic("用于判断的字符串不能为空")
	//}
	if strings.Index(facesStr, first) < strings.Index(facesStr, second){
		return true
	} else {
		return false
	}
}

// 判断当前牌是否比传进来的小
func (poker *Poker)smaller(otherP *Poker) bool {
	if strings.Index(facesStr, poker.face) < strings.Index(facesStr, otherP.face){
		return true
	} else {
		return false
	}
}

// 转成用于计算结果的扑克手牌组
func PokersToPokersStr(pokers []*Poker) (result []string) {
	for _, p := range pokers {
		if p != nil {
			result = append(result, p.toString())
		}else{
			result = append(result, "nil")
		}
	}
	return
}

// 扑克数组转字符串
func PokersToString(pokers []*Poker) string {
	tmp := ""
	for _, p := range pokers {
		if p != nil {
			tmp += p.toString()
		}else{
			tmp += "nil"
		}
	}
	return tmp
}

// 获取一张face的权重
func faceWeightMulti(face string, multi int) int {
	return (strings.Index(facesStr, face) + 1) * multi
}

// 复制该牌当作赖子的替牌
func (poker *Poker)bornALaiZi() *Poker {
	tmpP := new(Poker)
	tmpP.face = poker.face
	tmpP.color = poker.color
	tmpP.whole = poker.whole
	tmpP.bornByLz = true
	return tmpP
}

// 将遍历到的牌回调出去
type sortAppendPokerCb func(p *Poker)
// 将一张牌放入数组并排序
func sortAppendPoker(pokers []*Poker, newPoker *Poker, cb sortAppendPokerCb) []*Poker {
	for index, p := range pokers {
		if cb != nil{
			cb(p)
		}
		// 如果新的比旧的小就插入，否则就往后移动
		if newPoker.smaller(p) {
			//注意：保存后部剩余元素，必须新建一个临时切片，因为append就是对第一个元素做操作，不是new一个元素
			rear := append([]*Poker{}, pokers[index:]...)
			pokers = append(pokers[:index], newPoker)
			return append(pokers, rear...)
		}
	}
	// 如果在循环中判断插入了，则不会走到这里
	return append(pokers, newPoker)
}

// 将一张牌放入数组并排序
func sortAppendPokerWithoutSeem(pokers []*Poker, newPoker *Poker) []*Poker {
	for index, p := range pokers {
		// 如果重复则不做任何操作
		if p.face == newPoker.face {
			return pokers
		}
		// 如果新的比旧的小就插入，否则就往后移动
		if newPoker.smaller(p) {
			//注意：保存后部剩余元素，必须新建一个临时切片，因为append就是对第一个元素做操作，不是new一个元素
			rear := append([]*Poker{}, pokers[index:]...)
			pokers = append(pokers[:index], newPoker)
			return append(pokers, rear...)
		}
	}
	// 如果在循环中判断插入了，则不会走到这里
	return append(pokers, newPoker)
}

// 计算两张牌差多少步进，后边减前边，差1则代表是连着的，差2就代表需要一个赖子
func pokersDis(first string, second string) int {
	return strings.Index(facesStr, second) - strings.Index(facesStr, first)
}

// 在两组同花中选出大的那组
func biggerPokers(first []*Poker, second []*Poker) []*Poker {
	fLen := len(first)
	sLen := len(second)
	var smaller int
	var longgerPokers []*Poker
	if fLen < sLen {
		smaller = fLen
		longgerPokers = second
	} else {
		smaller = sLen
		longgerPokers = first
	}
	for smaller > -1 {
		tmpFirstP := first[smaller]
		tmpSecondP := second[smaller]
		// 先减，因为之前的长度没有减1
		smaller--
		if tmpFirstP.face == tmpSecondP.face {
			continue
		}else if tmpFirstP.smaller(tmpSecondP){
			return second
		}else{
			return first
		}
	}
	// 一样大就返回长的那个
	return longgerPokers
}

// 反转牌组(必须是5张牌)
func reversePokers(pokers []*Poker) []*Poker {
	return []*Poker{pokers[4], pokers[3], pokers[2], pokers[1], pokers[0]}
}

func NewPoker(face string, color string) *Poker {
	p := new(Poker)
	p.face = face
	p.color = color
	p.whole = face + color
	p.bornByLz = false
	return p
}

// 生成一副牌，主要用于随机生成牌型
func MakeDeckOfCards(withLz bool) []*Poker {
	faceLen := len(facesStr)
	colorLen := len(colorsStr)
	result := []*Poker{}
	for i := 0; i < faceLen; i++ {
		tmpF := string(facesStr[i])
		for j := 0; j < colorLen; j++ {
			tmpC := string(colorsStr[j])
			result = append(result, NewPoker(tmpF, tmpC))
		}
	}
	if withLz {
		result = append(result, NewPoker(laiZiStr, laiZiStr))
	}
	return result
}

// 遍历一组牌，拿最小顺子
func getSmallestSZFromPokers(pokers []*Poker) []*Poker {
	allHave := []*Poker{nil, nil, nil, nil, nil}
	for _, tmpP := range pokers {
		switch tmpP.face {
		case "A":
			allHave[0] = tmpP
		case "2":
			allHave[1] = tmpP
		case "3":
			allHave[2] = tmpP
		case "4":
			allHave[3] = tmpP
		case "5":
			allHave[4] = tmpP
		}
	}
	return allHave
}


// 该变赖子的值
func (poker *Poker)changeLzFace(toFace string) {
	poker.face = toFace
	poker.whole = toFace + poker.color
}