package msg_server

import (
	"encoding/binary"
	"fmt"
	"math"
)

// 前两byte作为type，接着8byte作为msg id
// todo 测试不同js中做wrap是否与go兼容，有没注意事项

func WrapMsg(mType int, mID int64, msg []byte) []byte {
	if mType < 0 || mType > math.MaxUint16 || mID < 0 {
		panic(fmt.Sprintf("invalid msg type: %v, mID: %v", mType, mID))
	}
	tb := make([]byte, 2)
	binary.BigEndian.PutUint16(tb, uint16(mType))
	idB := make([]byte, 8)
	binary.BigEndian.PutUint64(idB, uint64(mID))
	return append(append(tb[:], idB[:]...), msg...)
}

func UnWrapMsg(msg []byte) (int, int64, []byte) {
	// 2 + 8
	if len(msg) < 10 {
		return -1, -1, []byte{}
	}
	return int(binary.BigEndian.Uint16(msg[:2])), int64(binary.BigEndian.Uint64(msg[2:10])), msg[10:]
}