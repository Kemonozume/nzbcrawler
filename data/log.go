package data

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
)

type Log struct {
	Id      int64  `json:"id"`
	Level   string `json:"level"`
	Time    int64  `json:"time"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
}

type logwrap struct {
	Level string    `json:"level"`
	Msg   string    `json:"msg"`
	Time  time.Time `json:"time"`
}

type DBLog struct {
	DB *gorm.DB
}

func (d DBLog) Write(p []byte) (int, error) {
	logtmp := logwrap{}
	log1 := Log{}
	message := ""
	tag := ""

	err := json.Unmarshal(p, &logtmp)
	if err != nil {
		log.Println(err.Error())
		return len(p), nil
	}

	log.Printf("%s %s", logtmp.Level, logtmp.Msg)
	if strings.Contains(logtmp.Msg, "]") {
		msg := strings.Split(logtmp.Msg, "]")

		tag = msg[0][1:]
		message = msg[1][1:]
	} else {
		tag = "main"
		message = logtmp.Msg
	}

	log1.Level = logtmp.Level
	log1.Message = message
	log1.Tag = tag
	log1.Time = logtmp.Time.Unix()
	err = d.DB.Create(&log1).Error
	if err != nil {
		log.Printf(err.Error())
	}

	return len(p), nil
}
