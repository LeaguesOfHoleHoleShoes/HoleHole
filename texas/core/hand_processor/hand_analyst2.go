package hand_processor

import (
	"strings"
	"strconv"
)

// 整个分析者的运作步骤就是:
// 1.摸牌，对牌进行归类做预判
// 2.在处理完最后一张牌时根据预判设置策略链，避免执行不必要的操作
// 3.由于每条策略链是从大到小排的，因此一旦找到了大牌则立刻终止，得出牌型和权重

// 分析指导者（存放分类策略），要求最多遍历一次，且执行判断方法最少次就要得出牌型结果
// 对每一手牌分析都要new一个这个对象。保证多线程下不会相互影响
type Analyst2 struct {
	// 保存策略链，就是采取的判断策略，方便动态调整
	playChain []HandType
	// 保存下一个要执行的方法，-1表示结束，-2表示从头开始（-2为保留状态，应该不需要用）
	next HandType
	// 当前的扑克牌
	curPoker *Poker
	// 记录最原始的牌型，方便查错
	originPokers []*Poker
	// 保存结果牌
	checkedPokers []*Poker
	// 保存遍历过的扑克牌，并且已经排好序了（如果出现了相同的牌，则后边不做排序处理了）。最后一个是当前正在处理的牌
	tmpPokers []*Poker
	// 标记当前扑克是第几张
	curPokerIndex int
	// 保存总共有多少张扑克
	totalPokerCount int
	// 保存预判牌有多少张，主要是赖子牌不加到预判牌中，因此该数有可能比totalPokerCount小1
	tmpPokerCount int

	// ------------- 保存结果
	// 保存牌型结果
	handType HandType
	// 保存牌型权重
	handWeight int

	// ------------- 摸牌时在这里保存判断结果，用于最后一次判断牌型
	// 保存多组多张牌(主要是map效率太低,能不用则不用)数据:["A_3", "2_2"]
	// 这里的数据是没有排序的
	moreThanOnes []string
	// 标志是否含有A,方便判断A2345和皇家同花顺
	haveA bool
	// 标志是否含有赖子
	haveLz bool
	// 保存各花色同花牌，并且也是排好序了的
	sTH []*Poker
	hTH []*Poker
	dTH []*Poker
	cTH []*Poker
	// 保存顺子（去掉重复的排序好了的牌）
	shunZi []*Poker

	// 这三个的数据是排了序的
	// 保存所有的一对
	allYD []string
	// 保存所有的三条
	allTZ3 []string
	// 保存所有的四条
	allTZ4 []string
	// 是否有葫芦（包括了赖子情况）
	haveHL bool
	// 有否有四条（包括了赖子情况）
	haveTZ4 bool
}

func (analyst *Analyst2)defaultNext() {
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

// 摸一张牌,需要考虑是否要排序,并且在这里就计算好有几对对子及以上
func (analyst *Analyst2)touchAPoker(poker *Poker) {
	// 做分类等特殊操作
	analyst.classifyPoker(poker)
	// 如果是赖子都不用将它加入到排序中
	if poker.face == "z" {
		return
	}
	if len(analyst.tmpPokers) > 0 {
		seems := 0
		analyst.tmpPokers = sortAppendPoker(analyst.tmpPokers, poker,
			func(p *Poker) {
				if p.face == poker.face { // 计算有几张相同的牌,方便计算对子
					seems += 1
				}
			})
		if seems > 0 {
			analyst.refreshSeems(poker.face, seems + 1)
		}
	}else{
		analyst.tmpPokers = append(analyst.tmpPokers, poker)
	}
}

// 每次摸牌时要分类和做特殊操作
func (analyst *Analyst2)classifyPoker(poker *Poker) {
	analyst.originPokers = append(analyst.originPokers, poker)
	// 分类
	switch poker.color {
	case "s":
		analyst.sTH = sortAppendPoker(analyst.sTH, poker, nil)
	case "h":
		analyst.hTH = sortAppendPoker(analyst.hTH, poker, nil)
	case "d":
		analyst.dTH = sortAppendPoker(analyst.dTH, poker, nil)
	case "c":
		analyst.cTH = sortAppendPoker(analyst.cTH, poker, nil)
	}
	// 插入顺子，如果不是赖子才放进去
	if poker.face != laiZiStr {
		analyst.shunZi = sortAppendPokerWithoutSeem(analyst.shunZi, poker)
	}
	// 特殊检查
	if !analyst.haveA && poker.face == "A" {   // 是否有A
		analyst.haveA = true
	}
	if !analyst.haveLz && poker.face == "z" {  // 是否有赖子
		// 赖子牌不加到预判牌型中，因此这里要减一
		analyst.tmpPokerCount = analyst.totalPokerCount - 1
		analyst.haveLz = true
	}
}

// 插入一个face并排序
func sortAppendFace(faces []string, newFace string) []string {
	for index, f := range faces {
		// 如果新的比旧的小就插入，否则就往后移动
		if isFirstFaceSmall(newFace, f) {
			//注意：保存后部剩余元素，必须新建一个临时切片，因为append就是对第一个元素做操作，不是new一个元素
			rear := append([]string{}, faces[index:]...)
			faces = append(faces[:index], newFace)
			return append(faces, rear...)
		}
	}
	// 如果在循环中判断插入了，则不会走到这里
	return append(faces, newFace)
}

// 检查所有一对以上的牌，搜集所有成对以上的牌
func (analyst *Analyst2)checkMoreThanOnes() {
	// 顺便对这个做一次排序
	// 赖子不能在这里做判断，否则可能导致赖子重用及假葫芦的问题
	for _, str := range analyst.moreThanOnes {
		if strings.Contains(str, "_2"){
			analyst.allYD = sortAppendFace(analyst.allYD, string(str[0]))
		}else if strings.Contains(str, "_3"){
			analyst.allTZ3 = sortAppendFace(analyst.allTZ3, string(str[0]))
		}else if strings.Contains(str, "_4"){
			analyst.allTZ4 = sortAppendFace(analyst.allTZ4, string(str[0]))
		}
	}
	analyst.preCheckHaveHL()
	analyst.preCheckHaveTZ4()
}

// 预判是否有四条
func (analyst *Analyst2)preCheckHaveTZ4() {
	if analyst.haveLz {
		// 有赖子的情况下三条会变四条
		if len(analyst.allTZ3) > 0 || len(analyst.allTZ4) > 0 {
			analyst.haveTZ4 = true
		}
	}else{
		if len(analyst.allTZ4) > 0 {
			analyst.haveTZ4 = true
		}
	}
}

// 预判是否有葫芦
func (analyst *Analyst2)preCheckHaveHL() {
	if analyst.haveLz {
		// 有赖子的情况下，有三条的就算是4条了，不会做成葫芦
		if len(analyst.allYD) > 1 && len(analyst.allTZ3) == 0 && len(analyst.allTZ4) == 0 {
			analyst.haveHL = true
		}
	}else{
		ydLen := len(analyst.allYD)
		tz3Len := len(analyst.allTZ3)
		// 必须不是四条
		if len(analyst.allTZ4) == 0 {
			if ydLen > 0 && tz3Len > 0 {
				analyst.haveHL = true
			}else if ydLen == 0 && tz3Len > 1{
				// 如果没有对子，但是有一个以上三条也算葫芦
				analyst.haveHL = true
			}
		}
	}
}

// 根据摸牌的预判改变策略链
func (analyst *Analyst2)changePlayChain() {
	analyst.checkMoreThanOnes()
	// 如果是同花，在里边设置chain，随后直接return
	if analyst.playChainForTH() {
		return
	}
	// 有赖子就从大往下走即可（除非某些大的要过滤，小的不用过滤，因为拿到大的就不会往下走了）
	if analyst.haveLz {
		analyst.playCForUnderTHWithLZ()
	}else{
		analyst.playCForUnderTH()
	}
}

// 设置非同花不含赖子的策略链
func (analyst *Analyst2)playCForUnderTH() {
	if len(analyst.allTZ4) > 0 {
		analyst.playChain = append(analyst.playChain, HandOfST4)
	}
	if analyst.haveHL {
		analyst.playChain = append(analyst.playChain, HandOfHL)
	}
	analyst.playChain = append(analyst.playChain, HandOfSZ)
	if len(analyst.allTZ3) > 0 {
		analyst.playChain = append(analyst.playChain, HandOfST3)
	}
	ydLen := len(analyst.allYD)
	if ydLen > 1 {
		analyst.playChain = append(analyst.playChain, HandOfLD)
	}
	if ydLen > 0 {
		analyst.playChain = append(analyst.playChain, HandOfYD)
	}
	analyst.playChain = append(analyst.playChain, HandOfDZ)
}

// 设置非同花含赖子的策略链
func (analyst *Analyst2)playCForUnderTHWithLZ() {
	// 这里的顺序很重要！从大到小往里放
	// 四条
	if len(analyst.allTZ4) > 0 || len(analyst.allTZ3) > 0 {
		analyst.playChain = append(analyst.playChain, HandOfST4)
	}
	// 有赖子葫芦必须判断
	if analyst.haveHL {
		analyst.playChain = append(analyst.playChain, HandOfHL)
	}
	analyst.playChain = append(analyst.playChain, HandOfSZ)
	// 如果有三条则加赖子就成了四条了
	if len(analyst.allTZ3) == 0 && len(analyst.allYD) > 0 {
		analyst.playChain = append(analyst.playChain, HandOfST3)
	}
	// 如果有赖子则不可能有两条，因为那一对会被搞成3条，而不是两对
	// 如果有赖子则必然有一对
	analyst.playChain = append(analyst.playChain, HandOfYD)
}

// 设置至少是同花的策略链，返回是否设置了chain
func (analyst *Analyst2)playChainForTH() bool {
	thShouldHave := 5
	if analyst.haveLz {
		thShouldHave = 4
	}
	// 可能有同花
	if len(analyst.sTH) >= thShouldHave || len(analyst.hTH) >= thShouldHave || len(analyst.dTH) >= thShouldHave || len(analyst.cTH) >= thShouldHave {
		analyst.playChain = []HandType{HandOfTH}
		// 这两个bool判断时就包含了赖子情况
		if analyst.haveTZ4 {
			analyst.playChain = append(analyst.playChain, HandOfST4)
		}
		if analyst.haveHL {
			analyst.playChain = append(analyst.playChain, HandOfHL)
		}
		return true
	}
	return false
}

// 添加对子到统计表中(一个face出现了几张)
func (analyst *Analyst2)refreshSeems(face string, zhang int) {
	if zhang > 4 {
		panic("超过了四张一样的牌:" + face)
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
func (analyst *Analyst2)isLastOne() bool {
	if analyst.curPokerIndex == analyst.totalPokerCount - 1 {
		return true
	}else {
		return false
	}
}

// 牌型判断的回调方法，每手牌型的判断状态由Analyst2统一管理
type EnumCb2 func (analyst *Analyst2)

// 方法链处理遍历的Poker，要求最多遍历一次，且执行判断方法最少次就要得出牌型结果
// 该变量中的方法顺序必须和hand.go中的类型枚举顺序相同
var cbs2 = []EnumCb2{isDZ2, checkYD, checkLD, checkTZ3, checkSZ, checkTH, checkHL, checkTZ4}
var quanZhong2 = []int{10, 10 * 100, 1000 * 1000, 1000000 * 10000, 10000000000 * 100000}
// 获取默认的分析者
func DefaultAnalyst2() *Analyst2 {
	a := new(Analyst2)
	// 在摸牌时就要对牌型做预判，这样来得到策略链
	a.haveA = false
	a.haveLz = false
	a.haveHL = false
	a.haveTZ4 = false
	return a
}

// 在加载的时候就进行了判断，因此此处一般不调用
// 判断牌型及权重，传进来的可以是一个定制的分析者(方便改变分析策略)
func (hand *Hand)analysis2(analyst *Analyst2) {
	// 所有状态都要从该方法中生成，以保证可以支持多线程处理
	pokers := hand.pokers
	if analyst == nil {
		analyst = DefaultAnalyst2()
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

func (analyst *Analyst2)DoAnalysis(index int, poker *Poker) {
	analyst.doAnalysis(index, poker)
}

// 封装分析逻辑，方便各处调用
func (analyst *Analyst2)doAnalysis(index int, poker *Poker) {
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
		cbs2[analyst.next](analyst)
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
func isDZ2(analyst *Analyst2) {
	analyst.next = -1
	analyst.handType = HandOfDZ
	// 选出最大的5张牌，也就是排序后的牌的最后5张
	analyst.checkedPokers = analyst.tmpPokers[(analyst.totalPokerCount - 5):]
	// 设置权重
	analyst.weightOfDZ()
}

// 单张的权重
func (analyst *Analyst2)weightOfDZ() {
	weight := 0
	//fmt.Println("判断单张权重:", len(analyst.checkedPokers), ", ", pokersToString(analyst.checkedPokers))
	for index, p := range analyst.checkedPokers {
		// 后边的元素权重至少要比前边权重之和都要大
		weight += faceWeightMulti(p.face, quanZhong2[index])
	}
	analyst.handWeight = weight
}

// 判断一对
func checkYD(analyst *Analyst2) {
	if analyst.haveLz {
		// 有赖子则说明没有对子，只用找序列中最大的牌来当对子即可
		maxPoker := analyst.tmpPokers[(analyst.tmpPokerCount - 1)] // 还要减掉赖子
		lz := maxPoker.bornALaiZi()
		// 有赖子则说明没有对子，那么取最后4张即可
		analyst.checkedPokers = append(analyst.tmpPokers[(analyst.totalPokerCount - 5):], lz)
		// 保证权重计算是正确的
		analyst.moreThanOnes = []string{maxPoker.face}
	} else {
		analyst.ydWithoutLz()
	}
	analyst.next = -1
	analyst.handType = HandOfYD
	// 权重设置
	analyst.weightOfYD()
}
// 获取没有赖子的一对
func (analyst *Analyst2)ydWithoutLz() {
	// 从最后一张牌开始遍历取三张大单牌，再把一对取出来
	index := analyst.totalPokerCount - 1
	dzCount := 0  // 记录单张的数目
	ydCount := 0  // 记录对子获取的数目
	dz := string(analyst.moreThanOnes[0][0])
	for index > -1 {
		tmpP := analyst.tmpPokers[index]
		if tmpP.face == dz {
			analyst.checkedPokers = append(analyst.checkedPokers, tmpP)
			ydCount++
		}else {
			if dzCount < 3 {
				analyst.checkedPokers = append(analyst.checkedPokers, tmpP)
				dzCount++
			}
		}
		if ydCount + dzCount == 5 {
			break
		}
		index--
	}
	analyst.checkedPokers = reversePokers(analyst.checkedPokers)
}

// 一对的权重
func (analyst *Analyst2)weightOfYD() {
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
func checkLD(analyst *Analyst2) {
	// 有赖子绝对不可能是两对
	if !analyst.haveLz {
		// 找到最大的那两对和最大的单张，这里边不可能有3条和4条，有就不可能走到这里
		index := analyst.totalPokerCount - 1
		dzCount := 0  // 记录单张的数目
		ydCount := 0  // 记录对子获取的数目
		ydLen := len(analyst.allYD)
		dz1 := analyst.allYD[ydLen - 1]
		dz2 := analyst.allYD[ydLen - 2]
		for index > -1 {
			tmpP := analyst.tmpPokers[index]
			if tmpP.face == dz1 || tmpP.face == dz2 {
				analyst.checkedPokers = append(analyst.checkedPokers, tmpP)
				ydCount++
			}else {
				if dzCount < 1 {
					analyst.checkedPokers = append(analyst.checkedPokers, tmpP)
					dzCount++
				}
			}
			if ydCount + dzCount == 5 {
				break
			}
			index--
		}
		analyst.moreThanOnes = []string{dz1, dz2}
		analyst.next = -1
		analyst.handType = HandOfLD
		// 权重设置
		analyst.weightOfLD()
	}
}
// 两对权重，大对权重更大
func (analyst *Analyst2)weightOfLD() {
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
func checkTZ3(analyst *Analyst2) {
	if analyst.haveLz {
		analyst.tz3WithLz()
	}else {
		analyst.tz3WithoutLz()
	}
	analyst.next = -1
	analyst.handType = HandOfST3
	// 最后计算权重跟顺序有关
	analyst.checkedPokers = reversePokers(analyst.checkedPokers)
	// 权重设置
	analyst.weightOfST3()
}

// 没有赖子的三条
func (analyst *Analyst2)tz3WithoutLz() {
	dz := analyst.allTZ3[len(analyst.allTZ3) - 1]
	index := analyst.totalPokerCount - 1
	dzCount := 0  // 记录单张的数目
	ydCount := 0  // 记录对子获取的数目
	for index > -1 {
		tmpP := analyst.tmpPokers[index]
		if tmpP.face == dz {
			analyst.checkedPokers = append(analyst.checkedPokers, tmpP)
			ydCount++
		}else {
			if dzCount < 2 {
				analyst.checkedPokers = append(analyst.checkedPokers, tmpP)
				dzCount++
			}
		}
		if ydCount + dzCount == 5 {
			break
		}
		index--
	}
	// 保证统计权重的时候是正确的值
	analyst.moreThanOnes = []string{dz}
}

// 带赖子的三条
func (analyst *Analyst2)tz3WithLz() {
	// 如果有赖子则直接找对子中最大的即可
	dz := analyst.allYD[len(analyst.allYD) - 1]
	index := analyst.tmpPokerCount - 1
	dzCount := 0  // 记录单张的数目
	ydCount := 0  // 记录对子获取的数目
	haveBornLz := false
	for index > -1 {
		tmpP := analyst.tmpPokers[index]
		if tmpP.face == dz {
			if !haveBornLz {
				analyst.checkedPokers = append(analyst.checkedPokers, tmpP.bornALaiZi())
				ydCount++
				haveBornLz = true
			}
			analyst.checkedPokers = append(analyst.checkedPokers, tmpP)
			ydCount++
		}else {
			if dzCount < 2 {
				analyst.checkedPokers = append(analyst.checkedPokers, tmpP)
				dzCount++
			}
		}
		if ydCount + dzCount == 5 {
			break
		}
		index--
	}
	// 保证统计权重的时候是正确的值
	analyst.moreThanOnes = []string{dz}
}

// 三条的权重
func (analyst *Analyst2)weightOfST3() {
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
func checkSZ(analyst *Analyst2) {
	var resultShunZi []*Poker
	// 要用非重复的序列来遍历，而且最好是从最后往前找，这样才能找到最大的
	if analyst.haveLz {
		resultShunZi = checkSZNotFormatWithLz(analyst.shunZi)
	} else {
		resultShunZi = checkSZNotFormat(analyst.shunZi, analyst.haveA)
	}
	//fmt.Println("顺子判断结果：", pokersToString(resultShunZi))
	//fmt.Println("缓存牌:", pokersToString(analyst.tmpPokers))
	if resultShunZi != nil {
		analyst.checkedPokers = resultShunZi
		analyst.next = -1
		analyst.handType = HandOfSZ
		// 设置权重
		analyst.weightOfSZ()
	}
}
// 顺子的权重
func (analyst *Analyst2)weightOfSZ() {
	weight := 0
	// 只用看最后一张的大小即可
	if analyst.haveA {
		tmpFaces := ""
		for _, p := range analyst.tmpPokers {
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
func checkTH(analyst *Analyst2)  {
	// 如果不是同花顺，是同花顺在里边就设置了
	if !checkTHS(analyst) {
		var resultPokers []*Poker
		if analyst.haveLz {
			resultPokers = resultPokersForTHWithLz(analyst)
		}else{
			// 不是赖子就取出最大的那个同花即可
			resultPokers = biggerTHWithCheck(analyst.sTH, resultPokers, 5)
			resultPokers = biggerTHWithCheck(analyst.hTH, resultPokers, 5)
			resultPokers = biggerTHWithCheck(analyst.cTH, resultPokers, 5)
			resultPokers = biggerTHWithCheck(analyst.dTH, resultPokers, 5)
			rpLen := len(resultPokers)
			if rpLen > 5 {
				resultPokers = resultPokers[(rpLen - 5):rpLen]
			}
		}
		analyst.handType = HandOfTH
		//还需要判断四条和葫芦
		//analyst.next = -1
		analyst.checkedPokers = resultPokers
		analyst.weightOfTH()
	}
}

// 获取带赖子的最大同花
func resultPokersForTHWithLz(analyst *Analyst2) (resultPokers []*Poker) {
	resultPokers = biggerTHWithCheck(analyst.sTH, resultPokers, 4)
	resultPokers = biggerTHWithCheck(analyst.hTH, resultPokers, 4)
	resultPokers = biggerTHWithCheck(analyst.cTH, resultPokers, 4)
	resultPokers = biggerTHWithCheck(analyst.dTH, resultPokers, 4)
	rLen := len(resultPokers)
	if rLen > 4 { // 只取最大的四张，最后一张用赖子来替，这样才能保证牌是最大的
		resultPokers = resultPokers[rLen - 4:]
	}
	// 从大往小遍历找到没有的那张
	maxP := resultPokers[3]
	lzP := maxP.bornALaiZi()
	if maxP.face != "A" {
		lzP.changeLzFace("A")
	}else {
		// 找到没有的那张最大的
		for i := len(facesStr) - 1; i > -1 ; i-- {
			notFound := true
			tmpFaceStri := string(facesStr[i])
			for _, p := range resultPokers {
				if p.face == tmpFaceStri {
					notFound = false
					break
				}
			}
			if notFound {
				lzP.changeLzFace(tmpFaceStri)
				break
			}
		}
	}
	resultPokers = sortAppendPokerWithoutSeem(resultPokers, lzP)
	return
}

// 量身定制的方法
func biggerTHWithCheck(first []*Poker, second []*Poker, minLen int) []*Poker {
	if len(first) >= minLen {
		if second == nil {
			return first
		}else {
			return biggerPokers(first, second)
		}
	}
	return second
}

// 判断同花的权重，跟单张一样
func (analyst *Analyst2)weightOfTH() {
	analyst.weightOfDZ()
}

// 判断葫芦，这里只用加权重即可，进到这里就说明已经是葫芦了
func checkHL(analyst *Analyst2) {
	var dzB string
	var dzS string
	ydLen := len(analyst.allYD)
	if analyst.haveLz {
		// 有赖子找到最大的两对即可，不需要考虑三条，三条会变4条
		dzB = analyst.allYD[ydLen - 1]
		dzS = analyst.allYD[ydLen - 2]
	}else{
		// 如果没赖子则找三条中最大的和三条(第二大三条和第一大一对比)或一对中中最大的即可
		tz3Len := len(analyst.allTZ3)
		dzB = analyst.allTZ3[tz3Len - 1]
		if tz3Len > 1 && ydLen > 0 {
			tmpTZ3Second := analyst.allTZ3[tz3Len - 2]
			tmpYDFirst := analyst.allYD[ydLen - 1]
			if isFirstFaceSmall(tmpTZ3Second, tmpYDFirst) {
				dzS = tmpYDFirst
			}else {
				dzS = tmpTZ3Second
			}
		}else if ydLen == 0 {
			dzS = analyst.allTZ3[tz3Len - 2]
		}else {
			dzS = analyst.allYD[ydLen - 1]
		}
	}
	index := analyst.tmpPokerCount - 1
	bCount := 0  // 记录较大的牌的数量
	sCount := 0  // 记录较小的牌的数量
	haveBornLz := false
	// 前边判断了同花的话，结果可能已经被写入了同花
	analyst.checkedPokers = []*Poker{}
	for index > -1 {
		tmpP := analyst.tmpPokers[index]
		if tmpP.face == dzB {
			// 有赖子的情况下才去生成赖子牌
			if analyst.haveLz && !haveBornLz {
				analyst.checkedPokers = append(analyst.checkedPokers, tmpP.bornALaiZi())
				bCount++
				haveBornLz = true
			}
			analyst.checkedPokers = append(analyst.checkedPokers, tmpP)
			bCount++
		}else if tmpP.face == dzS {
			if sCount < 2 {
				analyst.checkedPokers = append(analyst.checkedPokers, tmpP)
			}
			sCount++

		}
		if bCount + sCount == 5 {
			break
		}
		index--
	}
	analyst.moreThanOnes = []string{dzB + "_3", dzS + "_2"}
	analyst.handType = HandOfHL
	analyst.next = -1
	// 设置权重
	analyst.weightOfHL()
}

// 葫芦权重
func (analyst *Analyst2)weightOfHL() {
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
func checkTZ4(analyst *Analyst2) {
	var bigStr string
	tz4Len := len(analyst.allTZ4)
	if analyst.haveLz {
		tz3Len := len(analyst.allTZ3)
		if tz3Len > 0 && tz4Len > 0 {
			tmp3 := analyst.allTZ3[tz3Len - 1]
			tmp4 := analyst.allTZ4[tz4Len - 1]
			if isFirstFaceSmall(tmp3, tmp4) {
				bigStr = tmp4
			}else{
				bigStr = tmp3
			}
		}else {
			if tz4Len > 0 {
				bigStr = analyst.allTZ4[tz4Len - 1]
			}else {
				bigStr = analyst.allTZ3[tz3Len - 1]
			}
		}
	}else {
		bigStr = analyst.allTZ4[tz4Len - 1]
	}
	index := analyst.tmpPokerCount - 1
	bCount := 0  // 记录较大的牌的数量
	dzCount := 0  // 记录较小的牌的数量
	haveBornLz := false
	// 前边判断了同花的话，结果可能已经被写入了同花
	analyst.checkedPokers = []*Poker{}
	for index > -1 {
		tmpP := analyst.tmpPokers[index]
		if tmpP.face == bigStr {
			// 有赖子的情况下才去生成赖子牌
			if analyst.haveLz && !haveBornLz {
				analyst.checkedPokers = append(analyst.checkedPokers, tmpP.bornALaiZi())
				bCount++
				haveBornLz = true
			}
			analyst.checkedPokers = append(analyst.checkedPokers, tmpP)
			bCount++
		}else {
			if dzCount < 1 {
				analyst.checkedPokers = append(analyst.checkedPokers, tmpP)
				dzCount++
			}
		}
		if bCount + dzCount == 5 {
			break
		}
		index--
	}
	analyst.moreThanOnes = []string{bigStr}
	analyst.handType = HandOfST4
	analyst.next = -1
	analyst.weightOfST4()
}
// 四条权重
func (analyst *Analyst2)weightOfST4() {
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
func checkTHS(analyst *Analyst2) (settedQz bool) {
	var resultPokers []*Poker
	if analyst.haveLz {
		resultPokers = biggerSZ(resultPokers, checkSZNotFormatWithLz(analyst.sTH))
		// 如果找到了皇家同花顺，就不用再找了
		if !isBiggestSZ(resultPokers) {
			resultPokers = biggerSZ(resultPokers, checkSZNotFormatWithLz(analyst.hTH))
		}
		if !isBiggestSZ(resultPokers) {
			resultPokers = biggerSZ(resultPokers, checkSZNotFormatWithLz(analyst.cTH))
		}
		if !isBiggestSZ(resultPokers) {
			resultPokers = biggerSZ(resultPokers, checkSZNotFormatWithLz(analyst.dTH))
		}
	}else {
		resultPokers = biggerSZ(resultPokers, checkSZNotFormat(analyst.sTH, analyst.haveA))
		// 如果找到了皇家同花顺，就不用再找了
		if !isBiggestSZ(resultPokers) {
			resultPokers = biggerSZ(resultPokers, checkSZNotFormat(analyst.hTH, analyst.haveA))
		}
		if !isBiggestSZ(resultPokers) {
			resultPokers = biggerSZ(resultPokers, checkSZNotFormat(analyst.cTH, analyst.haveA))
		}
		if !isBiggestSZ(resultPokers) {
			resultPokers = biggerSZ(resultPokers, checkSZNotFormat(analyst.dTH, analyst.haveA))
		}
	}
	if resultPokers != nil {
		settedQz = true
		analyst.next = -1
		analyst.handType = HandOfTHS
		analyst.checkedPokers = resultPokers
		if !checkHJTHS(analyst) {
			analyst.weightOfTHS()
		}
	}else{
		settedQz = false
	}
	return
}
// 判断是不是最大的顺子了
func isBiggestSZ(resultPokers []*Poker) bool {
	// "2345A"也是顺子，所以不能光从A来判断
	if resultPokers == nil || resultPokers[4].face != "A" || resultPokers[3].face != "K"{
		return false
	}else {
		return true
	}
}

// 返回较大的顺子
func biggerSZ(first []*Poker, second []*Poker) []*Poker {
	if first != nil && second != nil {
		if first[len(first) - 1].smaller(second[len(second) - 1]) {
			return second
		}else {
			return first
		}
	}else if first != nil {
		return first
	}else {
		return second
	}
}

// 同花顺权重，跟顺子权重算法一样
func (analyst *Analyst2)weightOfTHS() {
	analyst.weightOfSZ()
}

// 判断是否是皇家同花顺
func checkHJTHS(analyst *Analyst2) (settedQz bool) {
	if isBiggestSZ(analyst.checkedPokers) {
		analyst.next = -1
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
func (analyst *Analyst2)weightOfHJTHS() {
	analyst.handWeight = 0
}

// 计算顺子的算法
func cutSZForCheck(pokers []*Poker, stepPokersLen int, checkMethod func(ps []*Poker)(int, []*Poker)) []*Poker {
	//fmt.Println("all sz:", pokersToString(pokers))
	shunZiLen := len(pokers)
	// 首先要有5张或以上的牌才需要判断
	if shunZiLen >= stepPokersLen {
		tmpIndexLen := stepPokersLen
		index := shunZiLen
		startI := index - tmpIndexLen
		for startI >= 0 {
			tmpShunZi := pokers[startI:index]
			//fmt.Println("tmp sz:", pokersToString(tmpShunZi))
			// 这里就无法判断到A2345
			result, resultPokers := checkMethod(tmpShunZi)
			//fmt.Println("result:", result, ", index:", index)
			if result == -1 {  // 找到了
				return resultPokers
			}else {
				// 计算下一次开始的地方，自己画图就知道为啥是这个算法了
				index = index - (tmpIndexLen - result) + 1
				//fmt.Println("next index:", index)
				startI = index - tmpIndexLen
			}
		}
	}
	return nil
}

// 在有赖子的情况下找到最大的顺子
// 算法核心就是4个中不能出现大于2的缺口
func checkSZNotFormatWithLz(pokers []*Poker) []*Poker {
	result := cutSZForCheck(pokers, 4, checkIsSZ2WithLz)
	// 如果没找到结果但是又有A，那么就看有没有2345A这几个
	if result == nil && len(pokers) > 3 {
		result = haveSmallestSZWithLz(pokers)
	}
	return result
}

// 检查含赖子是否有最小的顺子
func haveSmallestSZWithLz(pokers []*Poker) []*Poker {
	// 检查含赖子的最小顺子
	allHave := getSmallestSZFromPokers(pokers)
	var lz *Poker
	var lzIndex int
	for index, p := range allHave {
		// 赖子没有使用的情况下就生出一个赖子
		if p == nil && lz == nil {
			lz = pokers[0].bornALaiZi()
			switch index {
			case 0:
				lz.changeLzFace("A")
			case 1:
				lz.changeLzFace("2")
			case 2:
				lz.changeLzFace("3")
			case 3:
				lz.changeLzFace("4")
			case 4:
				lz.changeLzFace("5")
			}
			lzIndex = index
		}else if p == nil && lz != nil {
			// 赖子用过了，但是还有位是空的
			return nil
		}
	}
	if lz != nil {
		rear := append([]*Poker{}, allHave[lzIndex + 1:]...)
		allHave = append(allHave[:lzIndex], lz)
		allHave = append(allHave, rear...)
	}
	return allHave
}

// 用来判断给来的带赖子牌组是不是顺子，给进来的牌一定要是从小到大排好序的
// 这里不用判断A2345，因为很有可能外边牌是2345678A，那么2345A永远不会进来
// 检查是否是顺子，返回断掉的index的前一个元素（外边是从后往前找，这样才能拿出最大的），外边再来就从该元素开始遍历，如果是顺子就返回-1
func checkIsSZ2WithLz(pokers []*Poker) (int, []*Poker) {
	// 传进来的就是四张牌
	totalLen := len(pokers)
	lzFace := ""
	var lzIndex int
	index := totalLen - 1
	// 在1的时候就会向后判断了，因此这里不用判断0位
	for index > 0 {
		smallerPoker := pokers[index - 1]
		biggerFace := pokers[index].face
		smallerFace := smallerPoker.face
		dis := pokersDis(smallerFace, biggerFace)
		if dis == 2 { //需要用赖子
			if lzFace != "" { // 赖子已经被用了
				return lzIndex, nil
			}else {
				lzFace = smallerPoker.nextFace()
				lzIndex = index - 1
			}
		} else if dis > 2 { // 根本不需要用赖子，已经断层
			return index - 1, nil
		}
		index--
	}
	maxPoker := pokers[totalLen - 1]
	lzP := maxPoker.bornALaiZi()
	if lzFace == "" {
		// 如果没有用赖子，则首选放在结尾，如果结尾是A，那么就放在开头
		if maxPoker.face == "A" {
			lzP.face = "T"
		}else {
			lzP.face = maxPoker.nextFace()
		}
	}else {
		lzP.face = lzFace
	}
	pokers = sortAppendPokerWithoutSeem(pokers, lzP)
	return -1, pokers
}

// 从任意排好序的牌组中找到顺子，返回找到的最大的顺子，如果没找到就返回nil
func checkSZNotFormat(pokers []*Poker, haveA bool) []*Poker {
	result := cutSZForCheck(pokers, 5, checkIsSZ2)
	// 如果没找到结果但是又有A，那么就看有没有2345A这几个
	if result == nil && haveA  && len(pokers) > 4 {
		result = haveSmallestSZ(pokers)
	}
	return result
}

// 检查是否有最小的顺子，并返回那个顺子，如果没有则返回空
func haveSmallestSZ(pokers []*Poker) []*Poker {
	allHave := getSmallestSZFromPokers(pokers)
	for _, p := range allHave {
		if p == nil {
			return nil
		}
	}
	return allHave
}

// 用来判断给来的牌组是不是顺子，给进来的牌一定要是从小到大排好序的
// 这里不用判断A2345，因为很有可能外边牌是2345678A，那么2345A永远不会进来
// 检查是否是顺子，返回断掉的index的前一个元素（外边是从后往前找，这样才能拿出最大的），外边再来就从该元素开始遍历，如果是顺子就返回-1
func checkIsSZ2(pokers []*Poker) (int, []*Poker) {
	//fmt.Println("判断顺子:", pokersToString(pokers))
	totalLen := len(pokers)
	for index, p := range pokers {
		// 最后一张不需要判断
		if index < totalLen - 1 {
			if pokers[index + 1].face != p.nextFace() {
				return index, nil
			}
		}
	}
	return -1, pokers
}


// 接口方法
func (analyst *Analyst2)oPokers() []*Poker {
	return analyst.originPokers
}

func (analyst *Analyst2)cPokers() []*Poker {
	return analyst.checkedPokers
}

func (analyst *Analyst2)setTotalPokerCount(tCount int) {
	analyst.tmpPokerCount = tCount
	analyst.totalPokerCount = tCount
}

func (analyst *Analyst2)SetTotalPokerCount(tCount int) {
	analyst.setTotalPokerCount(tCount)
}

func (analyst *Analyst2)hType() HandType {
	return analyst.handType
}

func (analyst *Analyst2)hWeight() int {
	return analyst.handWeight
}

func (analyst *Analyst2)ExportHand() *Hand {
	return analyst.exportHand()
}

func (analyst *Analyst2)exportHand() *Hand {
	hand := new(Hand)
	//fmt.Println("最终结果checkedPokers：", pokersToString(analyst.checkedPokers))
	//fmt.Println("原始牌型:", pokersToString(analyst.originPokers))
	// 要把赖子换回来
	if analyst.haveLz {
		for index, p := range analyst.checkedPokers {
			if p.bornByLz {
				tmp := analyst.checkedPokers
				rear := append([]*Poker{}, tmp[(index + 1):]...)
				tmp = append(tmp[:index], newLz())
				analyst.checkedPokers = append(tmp, rear...)
				break
			}
		}
	}
	hand.pokers = analyst.checkedPokers
	hand.handType = analyst.handType
	hand.weight = analyst.handWeight
	hand.originPokers = analyst.originPokers
	return hand
}
