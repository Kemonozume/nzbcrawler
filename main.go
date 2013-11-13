package main

import (
	"./mydb"
	"./town"
	"./webserv"
	_ "code.google.com/p/go-sqlite/go1/sqlite3"
	"database/sql"
	"github.com/coopernurse/gorp"
	logging "github.com/dvirsky/go-pylog/logging"
	"github.com/robfig/config"
)

const (
	DUMP = 0
)

func main() {

	//db access for releases
	db, err := sql.Open("sqlite3", "./release.db")
	if err != nil {
		panic(err.Error())
	}
	reldb := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}
	reldb.AddTableWithName(town.Release{}, "release").SetKeys(false, "Checksum").ColMap("Checksum").SetUnique(true).SetNotNull(true)
	reldb.CreateTablesIfNotExists()

	//db for logs
	//different database cause of locks with high log frequency
	dblog, err := sql.Open("sqlite3", "./logs.db")
	dbmap := &gorp.DbMap{Db: dblog, Dialect: gorp.SqliteDialect{}}
	dbmap.AddTableWithName(mydb.Log{}, "log").SetKeys(true, "Uid")
	dbmap.CreateTablesIfNotExists()
	logdb := mydb.DBLog{DB: dbmap}
	if err != nil {
		panic(err.Error())
	}

	logging.SetOutput(logdb)

	//read config file
	c, _ := config.ReadDefault("default.ini")

	//webserver
	serv := &webserv.Server{Config: c, RelDB: reldb, LogDB: dbmap}
	serv.Init()

}
