package texas

import (
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/msg_server"
	"github.com/LeaguesOfHoleHoleShoes/HoleHole/texas/abstracts"
)

// lru做用户数据缓存，并支持强制刷新
type rpcUserGetter struct {

}

func (getter *rpcUserGetter) GetUserByToken(token string) msg_server.AbsUser {
	panic("implement me")
}

func (getter *rpcUserGetter) GetUser(id string) abstracts.User {
	panic("implement me")
}

