package core

var TableLevels = map[int]TableLevel {
	1: { Xm: 10, BringIn: 4000, MinHave: 500 },
	2: { Xm: 100, BringIn: 4000 * 10, MinHave: 500 * 10 },
	3: { Xm: 1000, BringIn: 4000 * 100, MinHave: 500 * 100 },
}

type TableLevel struct {
	// 小盲下注多少
	Xm uint64
	// 用户每次必须带入多少筹码
	BringIn uint64
	// 至少有多少筹码才不会被踢出桌子
	MinHave uint64
}
