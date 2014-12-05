package data

import (
	"encoding/json"
	"log"
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
	Tag   string    `json:"tag"`
}

type DBLog struct {
	DB *gorm.DB
}

func (d DBLog) Write(p []byte) (int, error) {
	logtmp := logwrap{}
	log1 := Log{}

	err := json.Unmarshal(p, &logtmp)
	if err != nil {
		log.Println(err.Error())
		return len(p), nil
	}

	log.Printf("[%s] [%s] %s", logtmp.Level, logtmp.Tag, logtmp.Msg)

	log1.Level = logtmp.Level
	log1.Message = logtmp.Msg
	log1.Tag = logtmp.Tag
	log1.Time = logtmp.Time.Unix()
	err = d.DB.Create(&log1).Error
	if err != nil {
		log.Printf(err.Error())
	}

	return len(p), nil
}
