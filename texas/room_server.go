package texas

import (
	"errors"
	"go.uber.org/zap"
	"sync"
	"sync/atomic"
	"fmt"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/common/msg_server"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/texas/abstracts"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/texas/core"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/common/log"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/common/util"
)

func NewRoomServer(tableCount int, tableSeatCount int, tableLevel int, srvPort int) *RoomServer {
	r := &RoomServer{ totalSeat: tableSeatCount * tableCount, userGetter: &rpcUserGetter{} }
	r.wsServer = msg_server.NewWsServer(srvPort, r.userGetter, r)

	tables := make([]abstracts.Table, tableCount)
	for i := 0; i < tableCount; i++ {
		tl := core.TableLevels[tableLevel]
		if tl.Xm == 0 {
			panic(fmt.Sprintf("unknown table level: %v", tableLevel))
		}
		tables[i] = core.NewTable(i, tableSeatCount, tl, r.wsServer)
	}
	r.tables = tables
	return r
}

/*

完成game server

思考：要给各个chan提供缓存位置，否则整个room就是单线程在跑。问题是方法里边回阻塞等待结果。加缓存也无法解决这个问题。可能只能通过在room加锁来解决。


*/
type RoomServer struct {
	tables []abstracts.Table
	wsServer *msg_server.WsServer
	totalSeat int
	// 记录哪个用户在哪张桌子
	//users map[string]abstracts.Table
	users sync.Map

	userGetter *rpcUserGetter

	started uint32
}

func (r *RoomServer) Handle(uID string, msgType int, mID int64, msg []byte) error  {
	u := r.userGetter.GetUser(uID)
	if u == nil {
		log.L.Debug("call quick start, but can't find user", zap.String("uid", uID))
		return errors.New("can't find user: " + uID)
	}
	switch msgType {
	case abstracts.MsgTypeQuickStart:
		r.quickStart(abstracts.CommonMsg{ MsgID: mID, User: u })
	case abstracts.MsgTypeLeave:
		r.leave(abstracts.CommonMsg{ MsgID: mID, User: u })
	case abstracts.MsgTypeReady:
		r.ready(abstracts.CommonMsg{ MsgID: mID, User: u })
	case abstracts.MsgTypeGameAction:
		var gMsg abstracts.PlayerActionMsg
		if err := util.ParseJsonFromBytes(msg, &gMsg); err != nil {
			return err
		}
		gMsg.UserID = uID
		gMsg.MsgID = mID
		r.gameMsg(gMsg)
	}
	return nil
}

// 快速开始
func (r *RoomServer) quickStart(msg abstracts.CommonMsg) {
	user := msg.User
	// 如果他已经在某张桌子，则直接将该桌子的场景返回给客户端
	tmp, ok := r.users.Load(user.ID())
	if ok {
		t := tmp.(abstracts.Table)
		r.sendMsg(msg, abstracts.MsgTypeTableScene, t.GetScene(user.ID()))
		return
	}

	var toTable abstracts.Table = nil
	// 找到一张有位置的桌子坐下
	for _, t := range r.tables {
		if err := t.Enter(user); err == nil {
			toTable = t
			r.users.Store(user.ID(), t)
			break
		}
	}

	if toTable != nil {
		r.sendMsg(msg, abstracts.MsgTypeTableScene, toTable.GetScene(user.ID()))
	} else {
		r.sendErr(msg, "no more seat")
	}
}

func (r *RoomServer) leave(msg abstracts.CommonMsg) {
	user := msg.User
	tmp, ok := r.users.Load(user.ID())
	if !ok {
		r.sendErr(msg, "user not in any table")
		return
	}
	t := tmp.(abstracts.Table)

	// leave table
	if err := t.Leave(user); err != nil {
		r.sendErr(msg, err.Error())
		return
	}
	// leave room
	r.users.Delete(user.ID())

	r.sendSuccess(msg, "leave success")
}

func (r *RoomServer) ready(msg abstracts.CommonMsg) {
	user := msg.User
	tmp, ok := r.users.Load(user.ID())
	if !ok {
		r.sendErr(msg, "user not in any table")
		return
	}
	t := tmp.(abstracts.Table)

	if err := t.Ready(user); err != nil {
		r.sendErr(msg, err.Error())
		return
	}

	// 需要返回success？
}

func (r *RoomServer) gameMsg(msg abstracts.PlayerActionMsg) {
	tmp, ok := r.users.Load(msg.UserID)
	if !ok {
		r.wsServer.Send(msg.UserID, abstracts.MsgTypeErr, msg.MsgID, util.StringifyJsonToBytes(abstracts.ErrResp{ Info: "user not in any table" }))
		return
	}
	t := tmp.(abstracts.Table)

	if err := t.Do(msg); err != nil {
		r.wsServer.Send(msg.UserID, abstracts.MsgTypeErr, msg.MsgID, util.StringifyJsonToBytes(abstracts.ErrResp{ Info: err.Error() }))
		return
	}
}

// send success
func (r *RoomServer) sendSuccess(msg abstracts.CommonMsg, info string) {
	r.wsServer.Send(msg.User.ID(), abstracts.MsgTypeSuccess, msg.MsgID, util.StringifyJsonToBytes(abstracts.SuccessResp{ Info: info }))
}

// send err
func (r *RoomServer) sendErr(msg abstracts.CommonMsg, info string) {
	r.wsServer.Send(msg.User.ID(), abstracts.MsgTypeErr, msg.MsgID, util.StringifyJsonToBytes(abstracts.ErrResp{ Info: info }))
}

// send msg
func (r *RoomServer) sendMsg(msg abstracts.CommonMsg, mt int, data interface{}) {
	r.wsServer.Send(msg.User.ID(), mt, msg.MsgID, util.StringifyJsonToBytes(data))
}

func (r *RoomServer) startServer() error {
	go r.wsServer.Run()
	return nil
}

func (r *RoomServer) stopServer() error { return nil }

func (r *RoomServer) startTables() error {
	for _, t := range r.tables {
		if err := t.Start(); err != nil {
			return err
		}
	}
	return nil
}

func (r *RoomServer) stopTables() error {
	for _, t := range r.tables {
		if err := t.Stop(); err != nil {
			return err
		}
	}
	return nil
}

func (r *RoomServer) Start() error {
	if atomic.LoadUint32(&r.started) == 1 {
		return errors.New("room already started")
	}

	if atomic.CompareAndSwapUint32(&r.started, 0, 1) {
		// start tables
		r.startTables()
		// start server
		r.startServer()
	} else {
		log.L.Warn("start room atomic.CompareAndSwapUint32(&r.started... is false")
	}

	return nil
}

func (r *RoomServer) Stop() error {
	if atomic.LoadUint32(&r.started) == 0 {
		return errors.New("room not started")
	}

	if atomic.CompareAndSwapUint32(&r.started, 1, 0) {
		r.stopTables()
		r.stopServer()
	} else {
		log.L.Warn("stop room atomic.CompareAndSwapUint32(&r.started... is false")
	}

	return nil
}
