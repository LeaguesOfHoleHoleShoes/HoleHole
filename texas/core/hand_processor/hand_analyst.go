package hand_processor

import (
	"strings"
	"strconv"
)

// 分析指导者（存放分类策略），要求最多遍历一次，且执行判断方法最少次就要得出牌型结果
// 对每一手牌分析都要new一个这个对象。保证多线程下不会相互影响
type Analyst struct {
	// 保存策略链，就是采取的判断策略，方便动态调整
	playChain []HandType
	// 保存下一个要执行的方法，-1表示结束，-2表示从头开始（-2为保留状态，应该不需要用）
	next HandType
	// 当前的扑克牌
	curPoker *Poker
	// 记录最原始的牌型，方便查错
	originPokers []*Poker
	// 保存遍历过的扑克牌，并且已经排好序了（如果出现了相同的牌，则后边不做排序处理了）。最后一个是当前正在处理的牌
	checkedPokers []*Poker
	// 标记当前扑克是第几张
	curPokerIndex int
	// 保存总共有多少张扑克
	totalPokerCount int

	// ------------- 保存结果
	// 保存牌型结果
	handType HandType
	// 保存牌型权重
	handWeight int

	// ------------- 摸牌时在这里保存判断结果，用于最后一次判断牌型
	// 保存多组多张牌(主要是map效率太低,能不用则不用)数据:["A_3", "2_2"]
	moreThanOnes []string
	// 标志是否是同花(默认是false，因为在摸牌时只要判断有一张牌不一样则可以设置其为true)
	notTH bool
	// 标志是否含有A,方便判断A2345和皇家同花顺
	haveA bool
}

func (analyst *Analyst)defaultNext() {
	changedNext := false
	// 设置next
	for i, cData := range analyst.playChain {
		if cData == analyst.next {
			changedNext = true
			if i < (len(analyst.playChain) - 1) {
				analyst.next = analyst.playChain[i + 1]
			}else {
				// 这里说明当前的next就是最后一个方法了，因此需要结束
				analyst.next = -1
			}
			// 找到了就必须break，否则就会一直遍历下去
			break
		}
	}
	if !changedNext {
		panic("在设置默认next时，next并没有被正确设置")
	}
}

// 检查是否含有A
func (analyst *Analyst)checkHaveA(p *Poker) {
	if !analyst.haveA && p.face == "A" {
		analyst.haveA = true
	}
}

// 检查是否是同花
func (analyst *Analyst)checkTH(p *Poker) {
	// 这里不能用checkedPokers，因为排序会产生移位，可能第一个颜色一直移到最后，导致判断是同花
	if p.color != analyst.originPokers[0].color {
		analyst.notTH = true
	}
}

// 摸一张牌,需要考虑是否要排序,并且在这里就计算好有几对对子及以上
func (analyst *Analyst)touchAPoker(poker *Poker) {
	analyst.originPokers = append(analyst.originPokers, poker)
	if len(analyst.checkedPokers) > 0 {
		joined := false
		seems := 0
		for index, p := range analyst.checkedPokers{
			if p.face == poker.face { // 计算有几张相同的牌,方便计算对子
				seems += 1
			}
			// 如果新牌更小就放入,否则就继续
			if poker.smaller(p){
				joined = true
				tmp := analyst.checkedPokers
				//注意：保存后部剩余元素，必须新建一个临时切片，因为append就是对第一个元素做操作，不是new一个元素
				rear := append([]*Poker{}, tmp[index:]...)
				tmp = append(tmp[:index], poker)
				analyst.checkedPokers = append(tmp, rear...)
				break
			}
		}
		if seems > 0 {
			analyst.refreshSeems(poker.face, seems + 1)
		}
		// 如果没有放入,则说明这张牌就是最大的牌
		if !joined {
			analyst.checkedPokers = append(analyst.checkedPokers, poker)
		}
	}else{
		analyst.checkedPokers = append(analyst.checkedPokers, poker)
	}
	// 特殊检查
	analyst.checkTH(poker)
	analyst.checkHaveA(poker)
}

// 根据摸牌的预判改变策略链
func (analyst *Analyst)changePlayChain() {
	// 如果是同花
	if !analyst.notTH {
		analyst.playChain = []HandType{HandOfTH}
		return
	}
	moreThanLen := len(analyst.moreThanOnes)
	// 同花会在上边走，该长度为0则只用判断顺子和单张
	if moreThanLen == 0 {
		analyst.playChain = []HandType{HandOfSZ, HandOfDZ}
		return
	}
	if moreThanLen == 1 {
		analyst.playChain = []HandType{HandOfYD, HandOfST3, HandOfST4}
		return
	}
	if moreThanLen == 2 {
		analyst.playChain = []HandType{HandOfLD, HandOfST3}
		return
	}
}

// 添加对子到统计表中(一个face出现了几张)
func (analyst *Analyst)refreshSeems(face string, zhang int) {
	if zhang > 4 {
		panic("超过了四张一样的排:" + face)
	}
	tmp := face + "_"
	found := false
	for index, d := range analyst.moreThanOnes {
		if strings.Contains(d, tmp) {
			found = true
			analyst.moreThanOnes[index] = tmp + strconv.Itoa(zhang)
			break
		}
	}
	if !found{
		analyst.moreThanOnes = append(analyst.moreThanOnes, tmp + strconv.Itoa(zhang))
	}
}

// 判断是否是最后一张牌了
func (analyst *Analyst)isLastOne() bool {
	if analyst.curPokerIndex == analyst.totalPokerCount - 1 {
		return true
	}else {
		return false
	}
}

// 牌型判断的回调方法，每手牌型的判断状态由Analyst统一管理
type EnumCb func (analyst *Analyst)

// 方法链处理遍历的Poker，要求最多遍历一次，且执行判断方法最少次就要得出牌型结果
// 该变量中的方法顺序必须和hand.go中的类型枚举顺序相同
var cbs = []EnumCb{isDZ, isYD, isLD, isST3, isSZ, isTH, isHL, isST4}
var quanZhong = []int{10, 10 * 100, 1000 * 1000, 1000000 * 10000, 10000000000 * 100000}
// 获取默认的分析者
func DefaultAnalyst() *Analyst {
	a := new(Analyst)
	// 在摸牌时就要对牌型做预判，这样来得到策略链
	a.notTH = false
	a.haveA = false
	return a
}

// 在加载的时候就进行了判断，因此此处一般不调用
// 判断牌型及权重，传进来的可以是一个定制的分析者(方便改变分析策略)
func (hand *Hand)analysis(analyst *Analyst) {
	// 所有状态都要从该方法中生成，以保证可以支持多线程处理
	pokers := hand.pokers
	if analyst == nil {
		analyst = DefaultAnalyst()
		analyst.totalPokerCount = len(hand.pokers)
	}
	// 遍历一次所有扑克，判断出牌型及权重
	for index, p := range pokers{
		analyst.doAnalysis(index, p)
	}
	//可以将排好序的手牌替换掉未排序的手牌
	//将结果放到hand中
	hand.handType = analyst.handType
	hand.weight = analyst.handWeight
	hand.pokers = analyst.checkedPokers
	hand.originPokers = analyst.originPokers
}

// 封装分析逻辑，方便各处调用
func (analyst *Analyst)doAnalysis(index int, poker *Poker) {
	analyst.touchAPoker(poker)
	analyst.curPokerIndex = index
	if !analyst.isLastOne() {
		return
	}
	// 根据摸牌结束的结果改变策略链
	analyst.changePlayChain()
	// 每新摸一张牌都要从0开始执行策略链
	analyst.next = analyst.playChain[0]
	analyst.curPoker = poker
	// 插入其中需要考虑是否排序,摸牌的时候还要判断是否有一样的牌,如果有则记数,避免算法中再遍历记数
	count := 0
	// 如果不是-1则说明该链还要往下走
	for analyst.next != -1{
		// 缓存当前的nex
		curNext := analyst.next
		cbs[analyst.next](analyst)
		// 如果在方法内没有改变下个要执行的判断，则在这里将下个要执行的方法置成默认链中的下一个执行方法
		// 当然首先要排除掉已经被ignore了的方法
		if curNext == analyst.next {
			analyst.defaultNext()
		}
		if count > len(cbs) {
			panic("判断太多次了")
		}
		count += 1
	}
}

// ----------------------------- 链中的函数，结构都必须满足之前声明的回调函数的结构

// 判断单张，单张判断必须放策略链最后
func isDZ(analyst *Analyst) {
	analyst.next = -1
	analyst.handType = HandOfDZ
	// 设置权重
	analyst.weightOfDZ()
}
// 单张的权重
func (analyst *Analyst)weightOfDZ() {
	weight := 0
	for index, p := range analyst.checkedPokers {
		// 后边的元素权重至少要比前边权重之和都要大
		weight += faceWeightMulti(p.face, quanZhong[index])
	}
	analyst.handWeight = weight
}

// 判断一对
func isYD(analyst *Analyst) {
	moreThanOneLen := len(analyst.moreThanOnes)
	// 确定是否是一对
	if moreThanOneLen == 1 && strings.Contains(analyst.moreThanOnes[0], "_2"){
		analyst.next = -1
		analyst.handType = HandOfYD
		// 权重设置
		analyst.weightOfYD()
	}
}
// 一对的权重
func (analyst *Analyst)weightOfYD() {
	weight := 0
	// 一对肯定是第一个的第一个字符为大小
	dz := string(analyst.moreThanOnes[0][0])
	weight += faceWeightMulti(dz, quanZhong[4])
	singleIndex := 1
	for _, p := range analyst.checkedPokers {
		// 如果不等才做计算，因为之前已经算了对子
		if dz != p.face {
			weight += faceWeightMulti(p.face, quanZhong[singleIndex])
			singleIndex += 1
		}
	}
	analyst.handWeight = weight
}

// 判断两对
func isLD(analyst *Analyst) {
	moreThanOneLen := len(analyst.moreThanOnes)
	if moreThanOneLen > 1{
		hl := false
		for _, tmp := range analyst.moreThanOnes {
			if strings.Contains(tmp, "_3") {
				hl = true
			}
		}
		analyst.next = -1
		if hl {
			analyst.handType = HandOfHL
			// 这里边设置权重
			isHL(analyst)
		}else{
			analyst.handType = HandOfLD
			// 设置权重
			analyst.weightOfLD()
		}
	}
}
// 两对权重，大对权重更大
func (analyst *Analyst)weightOfLD() {
	weight := 0
	dz1 := string(analyst.moreThanOnes[0][0])
	dz2 := string(analyst.moreThanOnes[1][0])
	var bigger string
	var smaller string
	if strings.Index(facesStr, dz1) > strings.Index(facesStr, dz2) {
		bigger = dz1
		smaller = dz2
	}else {
		bigger = dz2
		smaller = dz1
	}
	weight += faceWeightMulti(bigger, quanZhong[4])
	weight += faceWeightMulti(smaller, quanZhong[3])
	for _, p := range analyst.checkedPokers {
		// 两个都不是则说明是那张单牌
		if p.face != dz1 && p.face != dz2 {
			weight += faceWeightMulti(p.face, quanZhong[2])
		}
	}
	analyst.handWeight = weight
}

// 判断三条
func isST3(analyst *Analyst) {
	moreThanOneLen := len(analyst.moreThanOnes)
	if moreThanOneLen > 0 {
		yd := false
		st := false
		for _, tmp := range analyst.moreThanOnes {
			if strings.Contains(tmp, "_3") {
				st = true
			}else if strings.Contains(tmp, "_2"){
				yd = true
			}
		}
		if yd && st {
			analyst.next = -1
			analyst.handType = HandOfHL
			// 在这里加权重
			isHL(analyst)
		}else if st {
			analyst.next = - 1
			analyst.handType = HandOfST3
			// 在这里加权重
			analyst.weightOfST3()
		}
	}
}
// 三条的权重
func (analyst *Analyst)weightOfST3() {
	weight := 0
	dz := string(analyst.moreThanOnes[0][0])
	weight += faceWeightMulti(dz, quanZhong[4])
	singleIndex := 1
	for _, p := range analyst.checkedPokers {
		// 如果不等才做计算，因为之前已经算了对子
		if dz != p.face {
			weight += faceWeightMulti(p.face, quanZhong[singleIndex])
			singleIndex += 1
		}
	}
	analyst.handWeight = weight
}

// 判断是否是顺子，在判断是否是顺子的每一次都需要对牌处理结果做排序
func isSZ(analyst *Analyst) {
	if checkIsSZ(analyst.checkedPokers){
		analyst.next = -1
		analyst.handType = HandOfSZ
		// 设置权重
		analyst.weightOfSZ()
	}
}
// 顺子的权重
func (analyst *Analyst)weightOfSZ() {
	weight := 0
	// 只用看最后一张的大小即可
	if analyst.haveA {
		tmpFaces := ""
		for _, p := range analyst.checkedPokers {
			tmpFaces += p.face
		}
		// 如果是该特殊顺子，其是最小的顺子，因此权重为0就行了
		if tmpFaces == "2345A" {
			analyst.handWeight = 0
			return
		}
	}
	p := analyst.checkedPokers[len(analyst.checkedPokers) - 1]
	weight += faceWeightMulti(p.face, 1)
	analyst.handWeight = weight
}

// 判断是否是同花
func isTH(analyst *Analyst)  {
	if len(analyst.moreThanOnes) == 0 {
		notSeem := false
		for _, p := range analyst.checkedPokers {
			if p.color != analyst.checkedPokers[0].color {
				notSeem = true
			}
		}
		// 如果没有不一样的花色则说明是同花
		if !notSeem {
			// 判断是同花了则不需要判断了，因为接下来的判断都通过直接调用q
			analyst.next = -1
			analyst.handType = HandOfTH
			// 这里直接调判断同花顺的方法，不用再出去执行一次循环
			if !isTHS(analyst){
				// 如果没有设置权重,则设置权重
				analyst.weightOfTH()
			}
		}
	}
}
// 判断同花的权重，跟单张一样
func (analyst *Analyst)weightOfTH() {
	analyst.weightOfDZ()
}

// 判断葫芦，这里只用加权重即可，进到这里就说明已经是葫芦了
func isHL(analyst *Analyst) {
	if analyst.handType != HandOfHL {
		panic("葫芦应该在外边判断好")
	}
	// 设置权重
	analyst.weightOfHL()
}

// 葫芦权重
func (analyst *Analyst)weightOfHL() {
	weight := 0
	var st string
	var dz string
	for _, str := range analyst.moreThanOnes {
		if strings.Contains(str, "_3") {
			st = string(str[0])
		}else if strings.Contains(str, "_2") {
			dz = string(str[0])
		}
	}
	weight += faceWeightMulti(st, quanZhong[4])
	weight += faceWeightMulti(dz, quanZhong[3])
	analyst.handWeight = weight
}

// 判断四条
func isST4(analyst *Analyst) {
	moreThanOneLen := len(analyst.moreThanOnes)
	if moreThanOneLen == 1 {
		tmp := analyst.moreThanOnes[0]
		if strings.Contains(tmp, "_4") {
			analyst.next = -1
			analyst.handType = HandOfST4
			// 设置权重
			analyst.weightOfST4()
		}
	}
}
// 四条权重
func (analyst *Analyst)weightOfST4() {
	tz := string(analyst.moreThanOnes[0][0])
	weight := faceWeightMulti(tz, quanZhong[4])
	for _, p := range analyst.checkedPokers {
		// 如果不等则说明是那张单牌
		if p.face != tz {
			weight += faceWeightMulti(p.face, quanZhong[3])
		}
	}
	analyst.handWeight = weight
}

// 判断同花顺，只能从同花那个方法中调用
func isTHS(analyst *Analyst) (settedQz bool) {
	if analyst.curPokerIndex != analyst.totalPokerCount - 1 {
		panic("没有遍历到最后一张，不应该判断是否是同花顺")
	}
	if analyst.handType != HandOfTH {
		panic("该牌型当前不是同花，不能判断是否是同花顺")
	}
	if checkIsSZ(analyst.checkedPokers) {
		analyst.handType = HandOfTHS
		// 如果没有设置权重则设置权重
		if !isHJTHS(analyst){
			// 设置权重
			settedQz = true
			analyst.weightOfTHS()
		}
	}
	settedQz = false
	return
}
// 同花顺权重，跟顺子权重算法一样
func (analyst *Analyst)weightOfTHS() {
	analyst.weightOfSZ()
}

// 判断是否是皇家同花顺
func isHJTHS(analyst *Analyst) (settedQz bool) {
	if analyst.haveA {
		analyst.handType = HandOfHJTHS
		// 设置权重
		settedQz = true
		analyst.weightOfHJTHS()
	}else{
		settedQz = false
	}
	return
}
// 皇家同花顺不用比，都一样的
func (analyst *Analyst)weightOfHJTHS() {
	analyst.handWeight = 0
}
// ----------------------------- 链中的函数，结构都必须满足之前声明的回调函数的结构

// 用来判断给来的牌组是不是顺子，给进来的牌一定要是从小到大排好序的
func checkIsSZ(pokers []*Poker) bool {
	totalLen := len(pokers)
	haveA := false
	// 特殊牌型2345A必定最后一张是A
	if pokers[totalLen - 1].face == "A" {
		haveA = true
	}
	tmpFaces := ""
	for index, p := range pokers {
		tmpFaces += p.face
		// 最后一张不需要判断
		if index < totalLen - 1 {
			// 3456A?
			// A是最后一张，因此如果是2345那么会一直遍历到A，形成2345A的牌型
			if pokers[index + 1].face != p.nextFace() && !haveA {
				return false
			}
		}
		// 有A还要做这个判断
		if haveA && (index == totalLen - 1) {
			if tmpFaces == "2345A" || tmpFaces == "TJQKA" {
				return true
			}else {
				return false
			}
		}
	}
	return true
}


// 接口方法
func (analyst *Analyst)oPokers() []*Poker {
	return analyst.originPokers
}

func (analyst *Analyst)cPokers() []*Poker {
	return analyst.checkedPokers
}

func (analyst *Analyst)setTotalPokerCount(tCount int) {
	analyst.totalPokerCount = tCount
}

func (analyst *Analyst)hType() HandType {
	return analyst.handType
}

func (analyst *Analyst)hWeight() int {
	return analyst.handWeight
}

func (analyst *Analyst)exportHand() *Hand {
	hand := new(Hand)
	hand.pokers = analyst.checkedPokers
	hand.handType = analyst.handType
	hand.weight = analyst.handWeight
	hand.originPokers = analyst.originPokers
	return hand
}


