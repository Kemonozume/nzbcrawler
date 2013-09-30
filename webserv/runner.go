package webserv

import (
	"./../ghost"
	"./../town"
	_ "fmt"
	log "github.com/dvirsky/go-pylog/logging"
	"time"
)

type Runner struct {
	Server *Server
	Cht    chan bool
	Chg    chan bool
}

type StatusRunner struct {
	Id      int `xorm:"pk not null unique"`
	Name    string
	Running bool
	NextRun string
}

func (r *Runner) Init() {
	r.Server.StatusDB.Mutex.Lock()
	if err := r.Server.StatusDB.Eng.CreateTables(&StatusRunner{}); err != nil {
		log.Error(err.Error())
	}
	r.Server.StatusDB.Eng.Insert(StatusRunner{0, "town", false, ""})
	r.Server.StatusDB.Eng.Insert(StatusRunner{1, "ghost", false, ""})
	r.Server.StatusDB.Mutex.Unlock()
}

func (r *Runner) checkTown() bool {
	if r.Server.Config2.TownName == "" {
		return false
	}
	if r.Server.Config2.TownPassword == "" {
		return false
	}
	return true
}

func (r *Runner) checkGhost() bool {
	if r.Server.Config2.GhostName == "" {
		return false
	}
	if r.Server.Config2.GhostPassword == "" {
		return false
	}
	return true
}

func (r *Runner) updateTime(id int, duration time.Duration) {
	//soon := time.Now().Add(duration)
	//r.Server.StatusDB.Mutex.Lock()
	//r.Server.StatusDB.Eng.Exec("update status_runner set next_run=? where id=?", fmt.Sprintf("next crawl: %v:%v", soon.Hour(), soon.Minute()), id)
	//r.Server.StatusDB.Mutex.Unlock()
}

func (r *Runner) Start() {
	go func() {
		timeout := time.Second * 1
		for {
			select {
			case <-time.After(timeout):
				if !r.checkTown() {
					timeout = time.Minute * 1
					log.Info("town: trying login again in %v minute", timeout)
				} else {
					tm := town.Townmanager{User: r.Server.Config2.TownName, Password: r.Server.Config2.TownPassword, DB: r.Server.RelDB, Status: r.Server.StatusDB}
					go tm.Start()
					c := time.Tick(time.Minute * 30)
					for _ = range c {
						log.Info("tick start town")
						go tm.Start()
					}
				}
			}
			break
		}
		return
	}()
	go func() {
		timeout := time.Second * 2
		for {
			select {
			case <-time.After(timeout):
				if !r.checkTown() {
					timeout = time.Minute * 1
					log.Info("ghost: trying to login again in %v minute", timeout)
				} else {
					gm := ghost.Ghostmanager{User: r.Server.Config2.GhostName, Password: r.Server.Config2.GhostPassword, DB: r.Server.RelDB, Status: r.Server.StatusDB}
					go gm.Start()
					c := time.Tick(time.Minute * 30)
					for _ = range c {
						log.Info("tick start ghost")
						go gm.Start()
					}
				}
			}
			break
		}
		return
	}()
}
