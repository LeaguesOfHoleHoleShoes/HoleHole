package mongo


import (
	"testing"
	"gopkg.in/mgo.v2/bson"
	"github.com/stretchr/testify/assert"
)

type person struct {
	Name  string
	Phone string
}

func TestInsertDb(t *testing.T) {
	hosts := []string{"localhost"}
	conf := NewDbConfig(hosts)

	db := GetDB(conf).DB("chooya")
	c := db.C("mytest")
	err := c.Insert(&person{"Lilei0", "18612345678"},
		&person{"Hanmeimei0", "18812345678"})
	if err != nil {
		panic(err)
	}
	//db.Session.Close()
	//ClearAllData(conf)
	CloseDb(conf)
}

func TestRemoveDB(t *testing.T){
	hosts := []string{"localhost"}
	conf := NewDbConfig(hosts)

	db := GetDB(conf).DB("chooya")
	c := db.C("mytest")
	err:=c.Remove(bson.M{"name":"Lilei0"})
	if err!=nil{
		assert.NoError(t,err)
	}
	CloseDb(conf)
}

