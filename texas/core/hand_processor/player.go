package hand_processor

import (
	"fmt"
	"strconv"
	"time"
	"runtime"
	"io/ioutil"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/util"
)

var coreNum = runtime.NumCPU()

// 开始游戏
func Play() {
	mode := inputMode()

	var startT time.Time
	var endT time.Time
	setEndT := false
	if mode == "2" {
		// 接收客户端输入的手牌，在获取的时候就归类，这样不用再全部遍历一次了
		//modeOfInputHand(hands)
		//fmt.Println("输入的手牌是：", hands)
		// 要在接收输入完成后开始计时
		var rHands *RHands
		ReadJsonFromFile("data/rank_hands.json", &rHands)
		startT = time.Now()
		rHands = rHands.sort()
		result := util.StringifyJsonToBytes(rHands)
		err := ioutil.WriteFile("data/my_rank_result.json", result, 0666)
		if err != nil {
			fmt.Println("写入文件错误:", err)
		}
	}else if mode == "3" {
		var matches *Matches
		startT = time.Now()
		//ReadJsonFromFile("data/match.json", &matches)
		//ReadJsonFromFile("data/seven_cards.json", &matches)
		//ReadJsonFromFile("data/match_test.json", &matches)
		ReadJsonFromFile("../seven_cards_with_ghost.result.json", &matches)
		mLen := len(matches.Matches)
		// 大概80为分界线，80个以下开线程去做的开销比直接算的开销更大
		if coreNum < 2 || mLen < 80 {
			doMatch(matches.Matches)
		}else{
			// 多核计算
			doMatchesByChan(matches)
		}
		//在写文件之前设置时间
		endT = time.Now()
		setEndT = true

		result := util.StringifyJsonToBytes(matches)
		err := ioutil.WriteFile("./my_result.json", result, 0666)
		if err != nil {
			fmt.Println("写入文件错误:", err)
		}
		//fmt.Println("一共:", mLen, "例，比较结果是：", string(result))
	}else{
		// 不能计入输入个数时的时间
		startT = time.Now()
		//fmt.Println("输入的个数是：", count)
	}
	if !setEndT {
		endT = time.Now()
	}
	fmt.Println("一共花费：", endT.Sub(startT))
}

// 输入手牌模式
func modeOfInputHand(hands map[HandType][]*Hand) {
	aHand := "init"
	// 输入手牌
	for aHand != "end" {
		aHand = inputHand()
		// 判断输入的手牌是否合格，在转换的时候就判断是否是合格的手牌了
		hand, _ := HandStrToHand(aHand)
		if hand != nil {
			HandToCollection(hand, hands)
		}else{
			fmt.Println("输入的手牌不合格")
		}
	}
}

// 输入随机个数
func inputRandCount() int {
	count := -1
	for count < 0 {
		fmt.Print("输入随机生成的个数:")
		randCount := inputLine()
		tmpCount, err := strconv.Atoi(randCount)
		if err == nil && tmpCount > 0{
			count = tmpCount
		}else{
			fmt.Println("输入有误，请重新输入")
		}
	}
	return count
}

// 输入一组手牌
func inputHand() string {
	fmt.Print("输入一组手牌（'end'为结束输入）:")
	return inputLine()
}

// 选择模式
func inputMode() string {
	fmt.Print("选择模式，1：随机生成手牌，2：排序，3：使用配置好的手牌（默认1）:")
	return inputLine()
}

// 接受一行输入
func inputLine() string {
	var hand string
	fmt.Scanln(&hand)
	return hand
}

// 从文件中读取json数据
func ReadJsonFromFile(path string, result interface{}) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("ReadFile: ", err.Error())
		return
	}
	if err := util.ParseJsonFromBytes(bytes, result); err != nil {
		fmt.Println("Unmarshal: ", err.Error())
		return
	}
}

type Matches struct {
	Matches []*Match `json:"matches"`
}

// 同步计算
func doMatch(matches []*Match) {
	for _, m := range matches {
		m.aliceHand, _ = HandStrToHand(m.Alice)
		m.bobHand, _ = HandStrToHand(m.Bob)
		// 都不为空则说明牌型没错
		if m.aliceHand != nil && m.bobHand != nil {
			m.match()
		}else{
			m.Result = -1
		}
	}
}

// 对match数组执行匹配操作，异步
func doMatchWithChan(matches []*Match, c chan int) {
	for _, m := range matches {
		m.aliceHand, _ = HandStrToHand(m.Alice)
		m.bobHand, _ = HandStrToHand(m.Bob)
		// 都不为空则说明牌型没错
		if m.aliceHand != nil && m.bobHand != nil {
			m.match()
		}else{
			m.Result = -1
		}
	}
	c <- 1
}

type Match struct {
	Alice string `json:"alice"`
	Bob string `json:"bob"`
	aliceHand *Hand
	bobHand *Hand
	Result int `json:"result"`
}

// 执行对比方法，得出对比结果
func (m *Match)match() {
	tmpResult := m.aliceHand.Match(m.bobHand)
	if tmpResult != m.Result {
		panic("有结果不一样")
	}
}
// 异步执行所有对比
func doMatchesByChan(matches *Matches) {
	mLen := len(matches.Matches)
	chans := []chan int{}
	step := mLen / coreNum
	for i := 0; i < coreNum; i++ {
		var tmp []*Match
		// 如果是最后一组，可能因为除不尽而丢掉之后的数据
		if i == coreNum - 1{
			tmp = matches.Matches[i * step:]
		}else{
			tmp = matches.Matches[i * step:(i + 1) * step]
		}
		c := make(chan int)
		chans = append(chans, c)
		go doMatchWithChan(tmp, c)
	}
	// 将在这里阻塞
	for _, tmpC := range chans {
		<- tmpC
	}
}

type RankHand struct {
	RHand string `json:"hand"`
	RScore int `json:"score"`
	hand *Hand
	Rank string `json:"rank"`
}

func (rHand *RankHand)setTypeStr() {
	switch rHand.hand.handType {
	case HandOfHJTHS:
		rHand.Rank = "皇家同花顺"
	case HandOfTHS:
		rHand.Rank = "同花顺"
	case HandOfST4:
		rHand.Rank = "四条"
	case HandOfHL:
		rHand.Rank = "葫芦"
	case HandOfTH:
		rHand.Rank = "同花"
	case HandOfSZ:
		rHand.Rank = "顺子"
	case HandOfST3:
		rHand.Rank = "三条"
	case HandOfLD:
		rHand.Rank = "两对"
	case HandOfYD:
		rHand.Rank = "一对"
	case HandOfDZ:
		rHand.Rank = "单张"
	}
}

type RHands struct {
	Hands []*RankHand `json:"hands"`
}

func (rHands *RHands)sort() *RHands {
	var resultHands []*RankHand
	resultRHands := new(RHands)
	tmpHands := make(map[HandType][]*RankHand)
	// 分类排序
	for _, rh := range rHands.Hands {
		rh.hand, _ = HandStrToHand(rh.RHand)
		rh.setTypeStr()
		tmpArr := tmpHands[rh.hand.handType]
		if tmpArr == nil {
			tmpArr = []*RankHand{}
		}
		inserted := false
		for index, t := range tmpArr {
			// 从大到小排序
			if rh.hand.weight > t.hand.weight {
				rear := append([]*RankHand{}, tmpArr[index:]...)
				tmpArr = append(tmpArr[:index], rh)
				tmpArr = append(tmpArr, rear...)
				inserted = true
				break
			}
		}
		if !inserted {
			tmpArr = append(tmpArr, rh)
		}
		tmpHands[rh.hand.handType] = tmpArr
	}
	resultHands = tmpHands[HandOfHJTHS]
	if resultHands == nil{
		resultHands = []*RankHand{}
	}
	resultHands = append(resultHands, tmpHands[HandOfTHS]...)
	resultHands = append(resultHands, tmpHands[HandOfST4]...)
	resultHands = append(resultHands, tmpHands[HandOfHL]...)
	resultHands = append(resultHands, tmpHands[HandOfTH]...)
	resultHands = append(resultHands, tmpHands[HandOfSZ]...)
	resultHands = append(resultHands, tmpHands[HandOfST3]...)
	resultHands = append(resultHands, tmpHands[HandOfLD]...)
	resultHands = append(resultHands, tmpHands[HandOfYD]...)
	resultHands = append(resultHands, tmpHands[HandOfDZ]...)
	rLen := len(resultHands)
	for index, rh := range resultHands {
		rh.RScore = rLen - index
	}
	resultRHands.Hands = resultHands
	return resultRHands
}