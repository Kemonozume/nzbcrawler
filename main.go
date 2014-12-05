package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	"log"

	"github.com/Kemonozume/nzbcrawler/config"
	"github.com/Kemonozume/nzbcrawler/data"
	"github.com/Kemonozume/nzbcrawler/runner"
	"github.com/Kemonozume/nzbcrawler/web"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

const TAG = "main"

var cpuprofile = flag.Bool("cprof", false, "enables or disables cpu profiling")
var memprofile = flag.Bool("mprof", false, "enables or disables memory profiling")
var cpucount = flag.Int("threads", 1, "number of threads to use")

func main() {
	flag.Parse()

	if *cpuprofile {
		f, err := os.Create("cpu.prof")
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(f)
	}

	runtime.GOMAXPROCS(*cpucount)
	log.Printf("%s starting nzbcrawler", TAG)
	logrus.SetFormatter(&logrus.JSONFormatter{})

	conf, err := config.Load("default.ini")
	if err != nil {
		logrus.WithField("tag", TAG).Fatalf("error loading config: %s", err.Error())
		return
	}

	login := fmt.Sprintf("%s:%s@/%s", conf.DBUser, conf.DBPassword, conf.DBName)
	dbmy, err := gorm.Open("mysql", login)
	if err != nil {
		logrus.WithField("tag", TAG).Fatalf("error connecting to db: %s", err.Error())
		return
	}

	dbmy.LogMode(false)

	dbmy.CreateTable(data.Log{})
	dbmy.CreateTable(data.Release{})
	dbmy.CreateTable(data.Tag{})

	dbmy.DB().Ping()
	dbmy.DB().SetMaxIdleConns(10)
	dbmy.DB().SetMaxOpenConns(100)

	logrus.SetOutput(data.DBLog{DB: &dbmy})

	exit := make(chan bool)

	if conf.Crawl {
		run, err := runner.NewRunner(&dbmy, conf)
		if err != nil {
			logrus.WithField("tag", TAG).Fatalf("error starting runner: %s", err.Error())
		}
		go run.Start(exit)
	}

	server := web.Server{}
	server.DB = &dbmy
	server.Config = conf

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func(s *web.Server) {
		for sig := range c {
			logrus.WithField("tag", TAG).Infof("captured %v, starting to shutdown", sig)
			if *memprofile {
				f, err := os.Create("mem.prof")
				if err != nil {
					log.Fatal(err)
				}
				pprof.WriteHeapProfile(f)
				f.Close()
			}
			s.Close()
		}
	}(&server)

	server.Init()

	close(exit)
	time.Sleep(time.Second * 1)
	dbmy.Close()
	if *cpuprofile {
		pprof.StopCPUProfile()
	}
	os.Exit(1)
}
