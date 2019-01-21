package msg_server

import (
	"testing"
	"fmt"
	"net/url"
	"github.com/gorilla/websocket"
	"time"
	"github.com/stretchr/testify/assert"
	"math/big"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/log"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/util"
)

const (
	playReqMsg = 0x1
	playRespMsg = 0x2
)

type playReq struct {
	Name string
}

type playResp struct {
	Balance *big.Int
}

var fakeUsers = map[string]*fakeUser{
	"1": { id: "1" },
	"2": { id: "2" },
	"3": { id: "3" },
	"4": { id: "4" },
	"5": { id: "5" },
}

type fakeUser struct { id string }

func (f *fakeUser) ID() string { return f.id }

type fakeUserGetter struct {}

func (f *fakeUserGetter) GetUserByToken(token string) AbsUser { return fakeUsers[token] }

type fakeMsgHandler struct {
	msgCount int
	server *WsServer
}

func (f *fakeMsgHandler) Handle(uID string, msgType int, mID int64, msg []byte) error {
	f.msgCount++
	switch msgType {
	case playReqMsg:
		var req playReq
		if err := util.ParseJsonFromBytes(msg, &req); err != nil {
			return err
		}
		log.L.Sugar().Debug("receive req", req)
		if f.msgCount == 1 {
			if f.server == nil {
				panic("f.server is nil")
			}
			f.server.Send(uID, playRespMsg, 1, util.StringifyJsonToBytes(playResp{ Balance: big.NewInt(122) }))
		}
	}
	return nil
}

// 测试正常连接可以收发消息
func TestNormalSeen(t *testing.T) {
	assert.Equal(t, MsgTypeHandShake + 1, playReqMsg)
	assert.Equal(t, MsgTypeHandShake + 2, playRespMsg)

	h := &fakeMsgHandler{}
	// start server
	server := NewWsServer(3333, &fakeUserGetter{}, h)
	h.server = server
	go func() {
		if err := server.Run(); err != nil {
			fmt.Println("server run err", err)
		}
	}()

	time.Sleep(time.Millisecond)

	u := url.URL{Scheme: "ws", Host: "localhost:3333", Path: "/msg"}
	var dialer *websocket.Dialer

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		panic(err)
	}
	// hand shake
	err = conn.WriteMessage(websocket.BinaryMessage, WrapMsg(MsgTypeHandShake, 1, util.StringifyJsonToBytes(HandShakeReq{ Token: "1" })))
	assert.NoError(t, err)
	err = conn.WriteMessage(websocket.BinaryMessage, WrapMsg(playReqMsg, 1, util.StringifyJsonToBytes(playReq{ Name: "alice" })))
	assert.NoError(t, err)
	err = conn.WriteMessage(websocket.BinaryMessage, WrapMsg(playReqMsg, 1, util.StringifyJsonToBytes(playReq{ Name: "bob" })))
	assert.NoError(t, err)
	err = conn.WriteMessage(websocket.BinaryMessage, WrapMsg(playReqMsg, 1, util.StringifyJsonToBytes(playReq{ Name: "cc" })))
	assert.NoError(t, err)

	mt, mb, err := conn.ReadMessage()
	assert.Equal(t, websocket.BinaryMessage, mt)
	assert.NoError(t, err)
	msgType, _, msgB := UnWrapMsg(mb)
	var resp playResp
	err = util.ParseJsonFromBytes(msgB, &resp)
	assert.NoError(t, err)
	assert.Equal(t, playRespMsg, msgType)
	assert.Equal(t, big.NewInt(122), resp.Balance)


	// 测试读不到消息时，ping pong msg在起作用，并且ReadMessage是不会读到ping pong msg的
	//mt, mb, err = conn.ReadMessage()
	//assert.NoError(t, err)
	//assert.Equal(t, websocket.PingMessage, mt)


	// 测试客户端close后服务器能正确remove peer
	//conn.Close()
	//time.Sleep(100 * time.Millisecond)
	//assert.Nil(t, server.peerSet.getPeer("1"))
	//assert.Equal(t, 0, int(server.peerSet.peerCount))
	//// 客户端关闭conn后读和写都会报错
	//_, _, err = conn.ReadMessage()
	//assert.Error(t, err)
	//err = conn.WriteMessage(websocket.BinaryMessage, WrapMsg(playReqMsg, util.StringifyJsonToBytes(playReq{ Name: "closed alice" })))
	//log.L.Debug("write closed conn", zap.Error(err))
	//assert.Error(t, err)


	// 测试移除peer后handlePeer中的for会结束
	//assert.Equal(t, 1, int(server.peerSet.peerCount))
	//server.peerSet.removePeer("1")
	//time.Sleep(100 * time.Millisecond)
	//assert.Nil(t, server.peerSet.getPeer("1"))
	//assert.Equal(t, 0, int(server.peerSet.peerCount))
	//// TODO 注意：服务器关闭了conn，写是不会报错的，只有读会报错，因此在读失败后要调用一次close
	//_, _, err = conn.ReadMessage()
	//assert.Error(t, err)
	//conn.Close()
	//err = conn.WriteMessage(websocket.BinaryMessage, WrapMsg(playReqMsg, util.StringifyJsonToBytes(playReq{ Name: "closed alice" })))
	//assert.Error(t, err)


	// 测试替换peer后前一个handlePeer的for会正确结束，且后一个peer是能够正确收发消息的
	//assert.Equal(t, 1, int(server.peerSet.peerCount))
	//nConn, _, err := dialer.Dial(u.String(), nil)
	//if err != nil {
	//	panic(err)
	//}
	//err = nConn.WriteMessage(websocket.BinaryMessage, WrapMsg(MsgTypeHandShake, util.StringifyJsonToBytes(HandShakeReq{ Token: "1" })))
	//assert.NoError(t, err)
	//time.Sleep(20 * time.Millisecond)
	//assert.Equal(t, 1, int(server.peerSet.peerCount))
	//err = nConn.WriteMessage(websocket.BinaryMessage, WrapMsg(playReqMsg, util.StringifyJsonToBytes(playReq{ Name: "alice" })))
	//assert.NoError(t, err)
	//// TODO 注意：服务器关闭了conn，写是不会报错的，只有读会报错，因此在读失败后要调用一次close
	//_, _, err = conn.ReadMessage()
	//assert.Error(t, err)
	//conn.Close()
	//err = conn.WriteMessage(websocket.BinaryMessage, WrapMsg(playReqMsg, util.StringifyJsonToBytes(playReq{ Name: "old alice" })))
	//assert.Error(t, err)
	//// 新conn发送的消息还是可以正常接收
	//err = nConn.WriteMessage(websocket.BinaryMessage, WrapMsg(playReqMsg, util.StringifyJsonToBytes(playReq{ Name: "alice x" })))
	//assert.NoError(t, err)


	// 测试客户端连上之后不发任何消息会被超时后断连
	//nConn, _, err := dialer.Dial(u.String(), nil)
	//if err != nil {
	//	panic(err)
	//}
	//_, _, err = nConn.ReadMessage()
	//log.L.Debug("read hand shake failed conn", zap.Error(err))
	//assert.Error(t, err)

	time.Sleep(100 * time.Millisecond)
}