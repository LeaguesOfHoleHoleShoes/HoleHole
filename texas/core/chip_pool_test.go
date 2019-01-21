package core

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

// 用户等量余额，全都all in，5个人，每人2000筹码，0位置是D
func TestChipPool1(t *testing.T) {
	tp := newTermChipPool()
	tp.bet(1, 1, 10, false)
	tp.bet(1, 2, 20, false)
	tp.bet(1, 3, 2000, true)
	tp.bet(1, 4, 2000, true)
	tp.bet(1, 0, 2000, true)
	// 第二轮
	tp.bet(2, 1, 1990, true)
	tp.bet(2, 2, 1980, true)
	assert.Equal(t, tp.playerTotalBetByChildPool(1), tp.playerTotalBetByRound(1))
	assert.Equal(t, tp.playerTotalBetByChildPool(2), tp.playerTotalBetByRound(2))
	assert.Equal(t, tp.playerTotalBetByChildPool(3), tp.playerTotalBetByRound(3))
	assert.Equal(t, tp.playerTotalBetByChildPool(4), tp.playerTotalBetByRound(4))
	assert.Equal(t, tp.playerTotalBetByChildPool(0), tp.playerTotalBetByRound(0))

	// 测试不同结果是否计算正确
	// 1号位玩家获胜
	r := tp.finalize([][]uint{ {1}, {2, 3}, {4, 0} })
	assert.Len(t, r, 1)
	assert.Equal(t, uint64(2000 * 5), r[1])
	// 1、2号玩家获胜
	r = tp.finalize([][]uint{ {1, 2}, {3}, {4, 0} })
	assert.Len(t, r, 2)
	assert.Equal(t, uint64(5000), r[1])
	assert.Equal(t, uint64(5000), r[2])
	// 1、2、3号玩家获胜
	r = tp.finalize([][]uint{ {1, 2, 3}, {4, 0} })
	assert.Len(t, r, 3)
	assert.Equal(t, uint64(3333), r[1])
	assert.Equal(t, uint64(3333), r[2])
	assert.Equal(t, uint64(3333), r[3])
	// 1、2、3、4号玩家获胜
	r = tp.finalize([][]uint{ {1, 2, 3, 4}, {0} })
	assert.Len(t, r, 4)
	assert.Equal(t, uint64(2500), r[1])
	assert.Equal(t, uint64(2500), r[2])
	assert.Equal(t, uint64(2500), r[3])
	assert.Equal(t, uint64(2500), r[3])
	// 0、1、2、3、4号玩家获胜
	r = tp.finalize([][]uint{ {1, 2, 3, 4, 0} })
	assert.Len(t, r, 5)
	assert.Equal(t, uint64(2000), r[1])
	assert.Equal(t, uint64(2000), r[2])
	assert.Equal(t, uint64(2000), r[3])
	assert.Equal(t, uint64(2000), r[4])
	assert.Equal(t, uint64(2000), r[0])
}

/*
用户等量余额，部分all in，其他人弃牌。5个人，每人2000筹码，0位置是D
*/
func TestChipPool2(t *testing.T) {
	tp := newTermChipPool()
	tp.bet(1, 1, 10, false)
	tp.bet(1, 2, 20, false)
	tp.bet(1, 3, 2000, true)
	tp.bet(1, 4, 2000, true)

	// 0、1、2弃牌，3获胜
	r := tp.finalize([][]uint{ {3}, {4} })
	assert.Len(t, r, 1)
	assert.Equal(t, uint64(4030), r[3])
	// 0、1、2弃牌，3、4获胜
	r = tp.finalize([][]uint{ {3, 4} })
	assert.Len(t, r, 2)
	assert.Equal(t, uint64(2015), r[3])
	assert.Equal(t, uint64(2015), r[4])
	// 1弃牌2跟着all in
	tp.bet(1, 2, 1980, true)
	// 0、1弃牌，2、3、4获胜
	r = tp.finalize([][]uint{ {2, 3, 4} })
	assert.Len(t, r, 3)
	// 2000 * 3 + 10
	assert.Equal(t, uint64(2003), r[2])
	assert.Equal(t, uint64(2003), r[3])
	assert.Equal(t, uint64(2003), r[4])
}

// 用户等量余额，部分all in 部分下注
func TestChipPool3(t *testing.T) {
	tp := newTermChipPool()
	tp.bet(1, 1, 10, false)
	tp.bet(1, 2, 20, false)
	tp.bet(1, 3, 500, false)
	tp.bet(1, 4, 2000, true)
	// 0、1、2弃牌，3号位下注，4号all in后3号弃牌，4号获胜
	r := tp.finalize([][]uint{ {4} })
	assert.Equal(t, uint64(2530), r[4])
}

// 用户等量余额，无人all in，部分下注
func TestChipPool4(t *testing.T) {
	tp := newTermChipPool()
	// 3号下注500,4号加注到1000,0号再加注到1500，1号跟，2号弃牌，3号跟，4号弃牌，桌面剩0、1、3
	tp.bet(1, 1, 10, false)
	tp.bet(1, 2, 20, false)
	tp.bet(1, 3, 500, false)
	tp.bet(1, 4, 1000, false)
	tp.bet(1, 0, 1500, false)
	tp.bet(1, 1, 1490, false)
	tp.bet(1, 3, 1000, false)
	// 第二轮，0号下注100,1弃牌，3号跟
	tp.bet(2, 0, 100, false)
	tp.bet(2, 3, 100, false)
	// pool应该只有一个，没有分池
	assert.Nil(t, tp.pool.nextPool)
	// 0获胜
	r := tp.finalize([][]uint{ {0} })
	assert.Equal(t, uint64(5720), r[0])
	// 0、3获胜
	r = tp.finalize([][]uint{ {0, 3} })
	assert.Equal(t, uint64(5720 / 2), r[0])
	assert.Equal(t, uint64(5720 / 2), r[3])
}

/*
用户余额不等量，余额多的人先all in
1、2用户带入2000，0带入1500，3带入500，4带入300
*/
func TestChipPool5(t *testing.T) {
	tp := newTermChipPool()
	tp.bet(1, 1, 10, false)
	tp.bet(1, 2, 20, false)
	tp.bet(1, 3, 20, false)
	tp.bet(1, 4, 20, false)
	tp.bet(1, 0, 20, false)
	tp.bet(1, 1, 10, false)
	// 0弃牌，1 all in，2 all in，3 all in，4 all in
	tp.bet(2, 1, 1980, true)
	tp.bet(2, 2, 1980, true)
	// 在这里会分池，是少筹码的后all in，因此调用splitByLess
	tp.bet(2, 3, 480, true)
	// 这里又会做一次分池
	tp.bet(2, 4, 280, true)
	assert.Equal(t, tp.playerTotalBetByChildPool(1), tp.playerTotalBetByRound(1))
	assert.Equal(t, tp.playerTotalBetByChildPool(2), tp.playerTotalBetByRound(2))
	assert.Equal(t, tp.playerTotalBetByChildPool(3), tp.playerTotalBetByRound(3))
	assert.Equal(t, tp.playerTotalBetByChildPool(4), tp.playerTotalBetByRound(4))
	assert.Equal(t, tp.playerTotalBetByChildPool(0), tp.playerTotalBetByRound(0))
	// 两次分池，所有池子总数：20 * 5 + 1980 * 2 + 480 +280 = 4820
	// 1号池应该是在4下注时分出来的。数量应该是300 * 4 + 20 = 1220，20为0位置
	p1 := tp.pool
	//fmt.Println("pool 1", p1.totalChip(), p1.total)
	p2 := p1.nextPool
	assert.NotNil(t, p2)
	// 2号池应该是在3号下注时分出来的。数量200 * 3 = 600
	//fmt.Println("pool 2", p2.totalChip(), p2.total)
	p3 := p2.nextPool
	assert.NotNil(t, p3)
	// 3号池就只有1、2两个用户，数量1500 * 2 = 3000
	//fmt.Println("pool 3", p3.totalChip(), p3.total)
	assert.Nil(t, p3.nextPool)

	// 4号获胜，3、2第二，1第三。4号赢1号池中的筹码，2、3号平分2号池中的筹码，2号赢3号池中的筹码
	r := tp.finalize([][]uint{ {4}, {2, 3}, {1} })
	assert.Equal(t, p1.totalChip(), r[4])
	assert.Equal(t, p2.totalChip() / 2 + p3.totalChip(), r[2])
	assert.Equal(t, p2.totalChip() / 2, r[3])
	// 3号获胜，4、2、1第二。3号赢1、2池，3号池1、2平分，4号没得分
	r = tp.finalize([][]uint{ {3}, {1, 2, 4} })
	assert.Equal(t, p1.totalChip() + p2.totalChip(), r[3])
	assert.Equal(t, p3.totalChip() / 2, r[1])
	assert.Equal(t, p3.totalChip() / 2, r[2])
	// 3、4获胜，3、4平分1号池，3赢得2号池，1赢得3号池
	r = tp.finalize([][]uint{ {3, 4}, {1}, {2} })
	assert.Equal(t, p1.totalChip() / 2, r[4])
	assert.Equal(t, p1.totalChip() / 2 + p2.totalChip(), r[3])
	assert.Equal(t, p3.totalChip(), r[1])
	// 2、3获胜，2、3平分1、2号池，2号赢3号池
	r = tp.finalize([][]uint{ {2, 3}, {4}, {1} })
	assert.Equal(t, (p1.totalChip() + p2.totalChip()) / 2, r[3])
	assert.Equal(t, (p1.totalChip() + p2.totalChip()) / 2 + p3.totalChip(), r[2])
	// 2、4获胜，2、4平分1号池，2号赢2、3号池
	r = tp.finalize([][]uint{ {2, 4}, {3}, {1} })
	assert.Equal(t, p1.totalChip() / 2, r[4])
	assert.Equal(t, p1.totalChip() / 2 + p2.totalChip() + p3.totalChip(), r[2])
	// 2、3、4获胜，2、3、4平分1号池，2、3平分2号池，2号赢3号池
	r = tp.finalize([][]uint{ {2, 3, 4}, {1} })
	assert.Equal(t, p1.totalChip() / 3, r[4])
	assert.Equal(t, p1.totalChip() / 3 + p2.totalChip() / 2, r[3])
	assert.Equal(t, p1.totalChip() / 3 + p2.totalChip() / 2 + p3.totalChip(), r[2])
	// 1、2获胜，1、2平分所有池
	r = tp.finalize([][]uint{ {1, 2}, {4}, {3} })
	totalChip := p1.totalChip() + p2.totalChip() + p3.totalChip()
	assert.Equal(t, totalChip / 2, r[1])
	assert.Equal(t, totalChip / 2, r[2])
	// 1、2、3获胜，1、2、3平分1、2号池子，1、2平分3号池
	r = tp.finalize([][]uint{ {1, 2, 3}, {4} })
	p12 := p1.totalChip() + p2.totalChip()
	assert.Equal(t, p12 / 3, r[3])
	assert.Equal(t, p12 / 3 + p3.totalChip() / 2, r[1])
	assert.Equal(t, p12 / 3 + p3.totalChip() / 2, r[2])
	// 1、2、4获胜，1、2、4平分1号池，1、2平分2、3号池
	r = tp.finalize([][]uint{ {1, 2, 4}, {3} })
	p23 := p2.totalChip() + p3.totalChip()
	assert.Equal(t, p1.totalChip() / 3, r[4])
	assert.Equal(t, p1.totalChip() / 3 + p23 / 2, r[1])
	assert.Equal(t, p1.totalChip() / 3 + p23 / 2, r[2])
	// 1、2、3、4获胜，1、2、3、4平分1号池，1、2、3平分2号池，1、2平分3号池。实际结果就是他们平分了0号的20个筹码并收回了本金
	r = tp.finalize([][]uint{ {1, 2, 3, 4} })
	assert.Equal(t, p1.totalChip() / 4, r[4])
	assert.Equal(t, p1.totalChip() / 4 + p2.totalChip() / 3, r[3])
	assert.Equal(t, p1.totalChip() / 4 + p2.totalChip() / 3 + p3.totalChip() / 2, r[2])
	assert.Equal(t, p1.totalChip() / 4 + p2.totalChip() / 3 + p3.totalChip() / 2, r[1])
	//fmt.Println(r)
}
// 从上边那个测试可以判断结果计算没啥问题，重点还是分池结果正确即可（每次池子每个人对应的下注数应该正确），因此下边不用详细测试结果计算

/*
用户余额不等量，余额少的人先all in
用户余额不等量，余额多少交叉all in，并且需要覆盖分池后往低池补筹码的情况
1、2用户带入2000，0带入1500，3带入500，4带入300
*/
func TestChipPool6(t *testing.T) {
	tp := newTermChipPool()
	tp.bet(1, 1, 10, false)
	tp.bet(1, 2, 20, false)
	tp.bet(1, 3, 20, false)
	// 少的人先all in
	tp.bet(1, 4, 300, true)
	// 后边的人需补齐1号池，随后才下后边的池子
	// 这里分池覆盖了往1号池中补筹码的情况
	tp.bet(1, 0, 600, false)
	tp.bet(1, 1, 590, false)
	tp.bet(1, 2, 580, false)
	// 这里有个分池，因此上边是多的人先all的情况，完成了交叉all in
	tp.bet(1, 3, 480, true)
	// 后边的人继续all in
	tp.bet(2, 0, 900, true)
	tp.bet(2, 1, 1400, true)
	tp.bet(2, 2, 1400, true)
	assert.Equal(t, tp.playerTotalBetByChildPool(1), tp.playerTotalBetByRound(1))
	assert.Equal(t, tp.playerTotalBetByChildPool(2), tp.playerTotalBetByRound(2))
	assert.Equal(t, tp.playerTotalBetByChildPool(3), tp.playerTotalBetByRound(3))
	assert.Equal(t, tp.playerTotalBetByChildPool(4), tp.playerTotalBetByRound(4))
	assert.Equal(t, tp.playerTotalBetByChildPool(0), tp.playerTotalBetByRound(0))
	// 第一个池子300 * 5 = 1500，第二个池子200 * 4 = 800，第三个池子1000 * 3 = 3000，第四个池子500 * 2 = 1000。一共6300
	p1 := tp.pool
	for i := 0; i < 5; i++ {
		assert.Equal(t, 300, int(p1.total[uint(i)]))
	}
	p2 := p1.nextPool
	assert.Equal(t, 200, int(p2.total[0]))
	assert.Equal(t, 200, int(p2.total[1]))
	assert.Equal(t, 200, int(p2.total[2]))
	assert.Equal(t, 200, int(p2.total[3]))
	p3 := p2.nextPool
	assert.Equal(t, 1000, int(p3.total[0]))
	assert.Equal(t, 1000, int(p3.total[1]))
	assert.Equal(t, 1000, int(p3.total[2]))
	p4 := p3.nextPool
	assert.Equal(t, 500, int(p4.total[1]))
	assert.Equal(t, 500, int(p4.total[2]))
	assert.Nil(t, p4.nextPool)
}

// 用户余额不等量，分多级由少到多逐步all in
// 0带入100,1带入200,2带入300,3带入400,4带入500
func TestChipPool9(t *testing.T) {
	tp := newTermChipPool()
	tp.bet(1, 1, 10, false)
	tp.bet(1, 2, 20, false)
	tp.bet(1, 3, 20, false)
	tp.bet(1, 4, 20, false)
	tp.bet(1, 0, 100, true)
	tp.bet(1, 1, 190, true)
	tp.bet(1, 2, 280, true)
	tp.bet(1, 3, 380, true)
	tp.bet(1, 4, 480, true)
	// 分了5个池子，无论谁获胜都要退4那100个筹码（4是不可能弃牌的，因为他全场下注最多因此他必定是最后那个人，没有弃牌机会）
	p1 := tp.pool
	for i := 0; i < 5; i++ {
		assert.Equal(t, 100, int(p1.total[uint(i)]))
	}
	p2 := p1.nextPool
	for i := 1; i < 5; i++ {
		assert.Equal(t, 100, int(p2.total[uint(i)]))
	}
	p3 := p2.nextPool
	for i := 2; i < 5; i++ {
		assert.Equal(t, 100, int(p3.total[uint(i)]))
	}
	p4 := p3.nextPool
	for i := 3; i < 5; i++ {
		assert.Equal(t, 100, int(p4.total[uint(i)]))
	}
	p5 := p4.nextPool
	for i := 4; i < 5; i++ {
		assert.Equal(t, 100, int(p5.total[uint(i)]))
	}
	assert.Nil(t, p5.nextPool)
}

// 用户余额不等量，分多级由多到少逐步all in
// 0带入500,1带入400,2带入300,3带入200,4带入100
func TestChipPool10(t *testing.T) {
	tp := newTermChipPool()
	tp.bet(1, 1, 10, false)
	tp.bet(1, 2, 20, false)
	tp.bet(1, 3, 20, false)
	tp.bet(1, 4, 20, false)
	tp.bet(1, 0, 500, true)
	tp.bet(1, 1, 390, true)
	tp.bet(1, 2, 280, true)
	tp.bet(1, 3, 180, true)
	tp.bet(1, 4, 80, true)
	assert.Equal(t, tp.playerTotalBetByChildPool(1), tp.playerTotalBetByRound(1))
	assert.Equal(t, tp.playerTotalBetByChildPool(2), tp.playerTotalBetByRound(2))
	assert.Equal(t, tp.playerTotalBetByChildPool(3), tp.playerTotalBetByRound(3))
	assert.Equal(t, tp.playerTotalBetByChildPool(4), tp.playerTotalBetByRound(4))
	assert.Equal(t, tp.playerTotalBetByChildPool(0), tp.playerTotalBetByRound(0))

	// 肯定有0,0带入最多
	p1 := tp.pool
	//fmt.Println(p1)
	for i := 0; i < 5; i++ {
		assert.Equal(t, 100, int(p1.total[uint(i)]))
	}
	p2 := p1.nextPool
	//fmt.Println(p2)
	for i := 0; i < 4; i++ {
		assert.Equal(t, 100, int(p2.total[uint(i)]))
	}
	p3 := p2.nextPool
	//fmt.Println(p3)
	for i := 0; i < 3; i++ {
		assert.Equal(t, 100, int(p3.total[uint(i)]))
	}
	p4 := p3.nextPool
	//fmt.Println(p4)
	for i := 0; i < 2; i++ {
		assert.Equal(t, 100, int(p4.total[uint(i)]))
	}
	p5 := p4.nextPool
	//fmt.Println(p5)
	for i := 0; i < 1; i++ {
		assert.Equal(t, 100, int(p5.total[uint(i)]))
	}
	assert.Nil(t, p5.nextPool)
}

// 有人all in后，剩下的人弃牌到只剩一个人
// 0带入2000,1带入1000,2、3、4带入1000
func TestChipPool11(t *testing.T) {
	tp := newTermChipPool()
	tp.bet(1, 1, 10, false)
	tp.bet(1, 2, 20, false)
	tp.bet(1, 3, 1000, true)
	// 4弃牌
	tp.bet(1, 0, 1500, false)
	// 1、2弃牌，剩下3和0
	// 应该有两个池子
	p1 := tp.pool
	assert.Equal(t, 10, int(p1.total[1]))
	assert.Equal(t, 20, int(p1.total[2]))
	assert.Equal(t, 1000, int(p1.total[3]))
	assert.Equal(t, 1000, int(p1.total[0]))
	p2 := p1.nextPool
	assert.Equal(t, 500, int(p2.total[0]))
	assert.Nil(t, p2.nextPool)
	// 3获胜赢第一个池子
	r := tp.finalize([][]uint{ {3}, {0} })
	assert.Equal(t, p1.totalChip(), r[3])
	assert.Equal(t, p2.totalChip(), r[0])
}

// 不all in，到最终比牌，该情况不应该出现分池
// 所有人都带入2000
func TestChipPool12(t *testing.T) {
	tp := newTermChipPool()
	tp.bet(1, 1, 10, false)
	tp.bet(1, 2, 20, false)
	tp.bet(1, 3, 1000, false)
	tp.bet(1, 4, 1000, false)
	tp.bet(1, 0, 1000, false)
	tp.bet(1, 1, 990, false)
	tp.bet(1, 2, 980, false)
	assert.Equal(t, tp.playerTotalBetByChildPool(1), tp.playerTotalBetByRound(1))
	assert.Equal(t, tp.playerTotalBetByChildPool(2), tp.playerTotalBetByRound(2))
	assert.Equal(t, tp.playerTotalBetByChildPool(3), tp.playerTotalBetByRound(3))
	assert.Equal(t, tp.playerTotalBetByChildPool(4), tp.playerTotalBetByRound(4))
	assert.Equal(t, tp.playerTotalBetByChildPool(0), tp.playerTotalBetByRound(0))

	assert.Equal(t, 1000 * 5, int(tp.pool.totalChip()))
	assert.Nil(t, tp.pool.nextPool)
}