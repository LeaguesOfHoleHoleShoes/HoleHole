package mongo

import (
	"gopkg.in/mgo.v2"
	"io/ioutil"
	"fmt"
	"strings"
	"time"
)

// new mongo conf
func NewDbConfig(hosts []string) *mgo.DialInfo {
	uname := "bee"
	pwd := "bee"
	pwdB, pwdErr := ioutil.ReadFile("/usr/local/.db/mongo.pas")
	unameB, unameErr := ioutil.ReadFile("/usr/local/.db/mongo.uname")
	if unameErr != nil { fmt.Println("读取mongo用户名文件出错:" + unameErr.Error()) }
	if pwdErr != nil { fmt.Println("读取mongo用户名文件出错:" + pwdErr.Error()) }
	if unameErr == nil && pwdErr == nil {
		uname = strings.TrimSpace(string(unameB))
		pwd = strings.TrimSpace(string(pwdB))
	}
	return &mgo.DialInfo{
		Addrs: hosts,
		// 数据库的指定需要仔细参悟  可以是admin
		Database: "chooya",
		Username:  uname,
		Password:  pwd,
		Direct:    false,
		Timeout:   time.Second * 5,
		PoolLimit: 300, // Session.SetPoolLimit
	}
}
