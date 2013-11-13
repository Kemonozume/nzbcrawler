package mydb

import (
	"fmt"
	"github.com/coopernurse/gorp"
	"strings"
)

type DBLog struct {
	DB *gorp.DbMap
}

//`xorm:"id pk not null autoincr"`
type Log struct {
	Uid       int64 `db:"id"`
	Message   string
	Lvl       string
	Line      string
	Timestamp string
	Date      string
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

	err := d.DB.Insert(log)
	if err != nil {
		fmt.Print(err.Error())
	}

	return 0, nil
}
