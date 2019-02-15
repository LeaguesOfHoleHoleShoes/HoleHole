package mongo

import (
	"gopkg.in/mgo.v2"
	"sync"
	"strings"
	"github.com/inconshreveable/log15"
)

var session *mgo.Session
var mutex sync.Mutex

// 获取数据库连接
// 外边传进来的Database要是其要连的db，这里会先尝试去连admin（设置readWriteAnyDatabase就在admin），如果不成功则可能是只针对对应的数据库做了授权，因此再尝试去连对应数据库，如果对应数据库也失败了，那么就是真的用户名和密码错误了
func GetDB(dbConfig *mgo.DialInfo) *mgo.Session {
	if session != nil {
		return session
	}
	mutex.Lock()
	defer mutex.Unlock()
	
	log15.Info("初始化mongo db session")
	tmpDbName := dbConfig.Database
	// 先用admin尝试
	dbConfig.Database = "admin"
	var err error
	if session, err = mgo.DialWithInfo(dbConfig); err != nil {
		// 出错的话有可能是管理员只给该用户开了该数据库的权限
		dbConfig.Database = tmpDbName
		if session, err = mgo.DialWithInfo(dbConfig); err != nil {
			panic("mongodb连接报错2:" + err.Error())
		}
	}
	session.SetMode(mgo.Strong, true)

	return session
}

// 清空某个数据库下的所有数据
func ClearAllData(dbConfig *mgo.DialInfo, dbName string) {
	if strings.Contains(dbName, "test") {
		// 获取连接
		tmpDB := GetDB(dbConfig).DB(dbName)
		cName, _ := tmpDB.CollectionNames()
		for _, cn := range cName {
			// DropCollection不会清除session中缓存的index。单元测试会反复调ClearAllData，直接会导致后边的测试无法migrate index
			//tmpDB.C(cn).DropCollection()
			if _, err := tmpDB.C(cn).RemoveAll(nil); err != nil {
				panic(err)
			}
		}
	} else {
		log15.Warn("非法操作！在非测试环境下调用了清空所有数据的方法")
	}
}

// 关闭连接
func CloseDb(dbConfig *mgo.DialInfo) {
	mutex.Lock()
	defer mutex.Unlock()
	if session != nil {
		session.Close()
		session = nil
	}
}

