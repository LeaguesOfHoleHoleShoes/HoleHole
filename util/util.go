package util

import (
	"github.com/json-iterator/go"
	"reflect"
	"crypto/rand"
	"go.uber.org/zap"
	"encoding/binary"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/log"
)

// 解析json字符串
func ParseJson(data string, result interface{}) error {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	return json.Unmarshal([]byte(data), result)
}

// json转字符串
func StringifyJson(data interface{}) string {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	b, _ := json.Marshal(&data)
	return string(b)
}

// 解析json bytes
func ParseJsonFromBytes(data []byte, result interface{}) error {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	return json.Unmarshal(data, result)
}

// json bytes转字符串
func StringifyJsonToBytes(data interface{}) []byte {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	b, _ := json.Marshal(&data)
	return b
}

func StringifyJsonToBytesWithErr(data interface{}) ([]byte, error) {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	b, err := json.Marshal(&data)
	return b, err
}

// 根据限定随机生成一个数字
func RandANum(limit int) int {
	rb := make([]byte, 4)
	if n, err := rand.Read(rb); n != 4 || err != nil {
		log.L.Error("read rand bytes failed", zap.Int("read num", n), zap.Error(err))
		return 0
	}
	// todo 测试高位符号位，以及边界问题
	return int(binary.BigEndian.Uint32(rb)) % limit
}

// struct slice copy to interface slice
func InterfaceSliceCopy(to, from interface{}) {
	toV := reflect.ValueOf(to)
	fromV := reflect.ValueOf(from)
	toLen := toV.Len()
	for i := 0; i < toLen; i++ {
		toV.Index(i).Set(fromV.Index(i))
	}
	return
}