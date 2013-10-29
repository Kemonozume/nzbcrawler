package main

import (
	"./mydb"
	"./webserv"
	_ "code.google.com/p/go-sqlite/go1/sqlite3"
	log "github.com/dvirsky/go-pylog/logging"
	"github.com/lunny/xorm"
	"github.com/robfig/config"
	"sync"
)

const (
	DUMP = 0
)

func main() {

	//db access for releases
	RelDB := &mydb.MyDB{}
	RelEng, err := xorm.NewEngine("sqlite3", "./release.db")
	if err != nil {
		panic(err.Error())
	}
	RelDB.Eng = RelEng
	RelDB.Mutex = &sync.Mutex{}
	RelDB.Eng.ShowSQL = false

	RelEng.Query("ALTER TABLE release ADD COLUMN hits int DEFAULT 0;")
	RelEng.Query("ALTER TABLE release ADD COLUMN rating int DEFAULT 0;")

	//db for logs
	//different database cause of locks with high log frequency
	LogDB := &mydb.MyDB{}
	LogEng, err := xorm.NewEngine("sqlite3", "./logs.db")
	if err != nil {
		panic(err.Error())
	}
	LogDB.Eng = LogEng
	LogDB.Mutex = &sync.Mutex{}
	LogDB.Eng.ShowSQL = false

	//setting log
	bla := mydb.DBLog{LogDB}

	LogDB.Mutex.Lock()
	if err := LogDB.Eng.CreateTables(&mydb.Log{}); err != nil {
		log.Error(err.Error())
	}
	LogDB.Mutex.Unlock()

	log.SetOutput(bla)

	//read config file
	c, _ := config.ReadDefault("default.ini")

	//webserver
	serv := &webserv.Server{Config: c, RelDB: RelDB, LogDB: LogDB}
	serv.Init()

}
