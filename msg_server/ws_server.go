package msg_server

import (
	"net/http"
	"github.com/gorilla/websocket"
	"fmt"
	"sync"
	"time"
	"errors"
	"go.uber.org/zap"
	"sync/atomic"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/log"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/util"
)

var upgrader = websocket.Upgrader{} // use default options

// msg type
const (
	MsgTypeHandShake = 0x0
)

const (
	sendMsgChanCache = 50
	maxPeerCount = 1000

	handShakeWait = 8 * time.Second

	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

type AbsUser interface {
	ID() string
}

type userGetter interface {
	GetUserByToken(token string) AbsUser
}

type msgHandler interface {
	Handle(uID string, msgType int, msgID int64, msg []byte) error
}

func NewWsServer(port int, userGetter userGetter, msgHandler msgHandler) *WsServer {
	return &WsServer {
		port: port,
		userGetter: userGetter,
		msgHandler: msgHandler,
		peerSet: newWsPeerSet(),
		sendMsgChan: make(chan *cMsg, sendMsgChanCache),
	}
}

type WsServer struct {
	port int

	userGetter userGetter
	msgHandler msgHandler

	peerSet *wsPeerSet

	sendMsgChan chan *cMsg
}

type cMsg struct {
	msgID int64
	uID string
	msgType int
	content []byte
}

func (s *WsServer) Run() error {
	go s.loop()
	http.HandleFunc("/msg", s.handlePeer)
	return http.ListenAndServe(fmt.Sprintf(":%v", s.port), nil)
}

func (s *WsServer) loop() {
	for tmp := range s.sendMsgChan {
		s.send(tmp)
	}
}

func (s *WsServer) handlePeer(w http.ResponseWriter, r *http.Request) {
	log.L.Debug("receive new peer", zap.String("remote addr", r.RemoteAddr))
	if s.peerSet.peerCount >= maxPeerCount {
		log.L.Warn("can't receive new peer, too many peers", zap.Int64("cur count", s.peerSet.peerCount), zap.Uint64("max count", maxPeerCount))
		return
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()

	c.SetReadLimit(maxMessageSize)
	// hand shake
	uID, err := s.handleShake(c)
	if uID == "" || err != nil {
		log.L.Debug("hand shake failed", zap.Error(err), zap.String("u id", uID))
		return
	}

	np := newWsPeer(uID, c)
	s.peerSet.addPeer(np)
	defer s.peerSet.removePeer(uID)
	if err := np.start(); err != nil {
		panic(err)
	}

	c.SetReadDeadline(time.Now().Add(pongWait))
	c.SetPongHandler(func(string) error {
		//log.L.Debug("receive pong msg", zap.String("uid", uID))
		c.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.L.Debug("read msg failed", zap.Error(err))
			return
		}
		if mt != websocket.BinaryMessage {
			log.L.Debug("receive invalid msg", zap.Int("msg type", mt))
			return
		}
		msgType, mID, msgB := UnWrapMsg(message)
		if err = s.msgHandler.Handle(uID, msgType, mID, msgB); err != nil {
			log.L.Error("handle msg failed", zap.Error(err))
			return
		}
	}
}

type HandShakeReq struct {
	// 可以考虑下发一个公钥，随后消息需要加解密传输
	Token string `json:"token"`
}

func (s *WsServer) handleShake(c *websocket.Conn) (string, error) {
	c.SetReadDeadline(time.Now().Add(handShakeWait))

	var req HandShakeReq
	mt, mb, err := c.ReadMessage()
	if err != nil {
		return "", err
	}
	if mt != websocket.BinaryMessage {
		return "", errors.New(fmt.Sprintf("invalid msg type: %v", mt))
	}

	msgType, _, msgB := UnWrapMsg(mb)
	if msgType != MsgTypeHandShake {
		return "", errors.New(fmt.Sprintf("msg type isn't MsgTypeHandShake, %v", msgType))
	}
	if err = util.ParseJsonFromBytes(msgB, &req); err != nil {
		return "", err
	}
	if req.Token == "" {
		return "", errors.New("empty token")
	}

	u := s.userGetter.GetUserByToken(req.Token)
	if u == nil {
		return "", errors.New("invalid token")
	}
	log.L.Debug("hand shake success", zap.String("u id", u.ID()))
	return u.ID(), nil
}

func (s *WsServer) Send(id string, msgType int, msgID int64, msg []byte) {
	s.sendMsgChan <- &cMsg{ msgID: msgID, uID: id, msgType: msgType, content: msg }
}

func (s *WsServer) send(msg *cMsg) {
	p := s.peerSet.getPeer(msg.uID)
	if p == nil {
		log.L.Warn("can't find peer in peer set, msg not send", zap.String("uid", msg.uID))
		return
	}
	// 如果send失败，则会导致peer直接stop，接着就触发conn.close，那么这时上边的ReadMsg会read出err，此次连接的生命周期就此结束
	p.send(msg)
}

func newWsPeerSet() *wsPeerSet {
	return &wsPeerSet{}
}

type wsPeerSet struct {
	// key player id
	peers     sync.Map
	peerCount int64
}

func (ps *wsPeerSet) getPeer(id string) *wsPeer {
	if p, ok := ps.peers.Load(id); ok {
		return p.(*wsPeer)
	}
	return nil
}

func (ps *wsPeerSet) removePeer(id string) {
	if p := ps.getPeer(id); p != nil {
		log.L.Debug("remove peer", zap.String("uid", id))
		p.stop()
		ps.peers.Delete(id)
		atomic.AddInt64(&ps.peerCount, -1)
	}
}

func (ps *wsPeerSet) addPeer(p *wsPeer) {
	if preP := ps.getPeer(p.id); preP != nil {
		// Close后会触发remove，执行一次count-1
		preP.stop()
		// 等待上一个conn从peer set中删除
		time.Sleep(10 * time.Millisecond)
	}
	atomic.AddInt64(&ps.peerCount, 1)

	ps.peers.Store(p.id, p)
}

func newWsPeer(id string, conn *websocket.Conn) *wsPeer {
	return &wsPeer{
		id: id, conn: conn,
		sendChan: make(chan *cMsg, sendMsgChanCache),
	}
}

type wsPeer struct {
	// user id
	id string
	conn *websocket.Conn
	sendChan chan *cMsg
	stopChan chan struct{}
}

func (p *wsPeer) start() error {
	if p.stopChan != nil {
		return errors.New("peer already started")
	}
	p.stopChan = make(chan struct{})
	go p.loop()

	return nil
}

// close stop chan 后会调用conn.close
func (p *wsPeer) stop() error {
	if p.stopChan == nil {
		return errors.New("peer not started")
	}
	close(p.stopChan)
	p.stopChan = nil

	return nil
}

func (p *wsPeer) loop() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		p.conn.Close()
	}()
	for {
		select {
		case msg := <- p.sendChan:
			if err := p.doSend(msg); err != nil {
				return
			}

		case <- ticker.C:
			//log.L.Debug("send ping msg to", zap.String("uid", p.id))
			if err := p.doSend(&cMsg{ msgType: websocket.PingMessage }); err != nil {
				return
			}

		case <- p.stopChan:
			log.L.Debug("peer loop returned", zap.String("uid", p.id))
			return
		}
	}
}

func (p *wsPeer) send(msg *cMsg) {
	select {
	case p.sendChan <- msg:
	default:
		log.L.Warn("can't send msg to client", zap.String("uid", p.id), zap.Int("send chan len", len(p.sendChan)))
		// todo add retry in server
	}
}

func (p *wsPeer) doSend(msg *cMsg) error {
	p.conn.SetWriteDeadline(time.Now().Add(writeWait))

	mb := msg.content
	if msg.msgType == websocket.BinaryMessage {
		mb = WrapMsg(msg.msgType, msg.msgID, msg.content)
	}

	return p.conn.WriteMessage(msg.msgType, mb)
}