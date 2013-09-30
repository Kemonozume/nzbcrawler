package mydb

import (
	"fmt"
	"github.com/lunny/xorm"
	"strings"
	"sync"
)

type DBLog struct {
	DB *MyDB
}

type Log struct {
	Uid                                 int64 `xorm:"id pk not null autoincr"`
	Message, Lvl, Line, Timestamp, Date string
}

func (d DBLog) Write(p []byte) (int, error) {

	str := string(p)
	//n := len(str)
	astr := strings.Split(str, " ")

	log := &Log{}
	log.Date = astr[0]
	log.Timestamp = astr[1]
	log.Lvl = astr[2]
	log.Line = astr[4]
	log.Message = strings.Join(astr[5:], " ")

	//d.DB.Mutex.Lock()
	//d.DB.Eng.Insert(log)
	//d.DB.Mutex.Unlock()

	fmt.Print(str)

	return 0, nil
}

type MyDB struct {
	Eng   *xorm.Engine
	Mutex *sync.Mutex
}
