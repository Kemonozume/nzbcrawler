package webserv

import (
	"./../ghost"
	"./../town"
	"errors"
	"fmt"
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

func (r *Runner) checkTown() error {
	if r.Server.Config2.TownName == "" {
		return errors.New("no town login information (Name missing)")
	}
	if r.Server.Config2.TownPassword == "" {
		return errors.New("no town login information (Password missing)")
	}
	return nil
}

func (r *Runner) checkGhost() error {
	if r.Server.Config2.GhostName == "" {
		return errors.New("no ghost login information (Name missing)")
	}
	if r.Server.Config2.GhostPassword == "" {
		return errors.New("no ghost login information (Password missing)")
	}
	return nil
}

func (r *Runner) updateTime(id int, duration time.Duration) {
	soon := time.Now().Add(duration)
	r.Server.StatusDB.Mutex.Lock()
	r.Server.StatusDB.Eng.Exec("update status_runner set next_run=? where id=?", fmt.Sprintf("next crawl: %v:%v", soon.Hour(), soon.Minute()), id)
	r.Server.StatusDB.Mutex.Unlock()
}

func (r *Runner) Start() {

	go func() {
		timeout := time.Second * 1
		for {
			select {
			case <-time.After(timeout):
				err := r.checkTown()
				if err != nil {
					timeout = time.Minute * 1
					r.updateTime(0, timeout)
					log.Info(err.Error())
					log.Info("town: trying login again in %v minute", timeout)
				} else {
					tm := town.Townmanager{User: r.Server.Config2.TownName, Password: r.Server.Config2.TownPassword, DB: r.Server.RelDB, Status: r.Server.StatusDB}
					go tm.Start()
					c := time.Tick(time.Minute * 45)
					for _ = range c {
						r.updateTime(0, time.Minute*45)
						tm = town.Townmanager{User: r.Server.Config2.TownName, Password: r.Server.Config2.TownPassword, DB: r.Server.RelDB, Status: r.Server.StatusDB}
						go tm.Start()
					}
				}
			}
			break
		}
		return
	}()
	go func() {
		timeout := time.Second * 1
		for {
			select {
			case <-time.After(timeout):
				err := r.checkTown()
				if err != nil {
					timeout = time.Minute * 1
					r.updateTime(1, timeout)
					log.Info(err.Error())
					log.Info("ghost: trying to login again in %v minute", timeout)
				} else {
					gm := ghost.Ghostmanager{User: r.Server.Config2.GhostName, Password: r.Server.Config2.GhostPassword, DB: r.Server.RelDB, Status: r.Server.StatusDB}
					go gm.Start()
					c := time.Tick(time.Minute * 45)
					for _ = range c {
						r.updateTime(1, time.Minute*45)
						gm = ghost.Ghostmanager{User: r.Server.Config2.GhostName, Password: r.Server.Config2.GhostPassword, DB: r.Server.RelDB, Status: r.Server.StatusDB}
						go gm.Start()
					}
				}
			}
			break
		}
		return
	}()
}

/*
func (r *Runner) Start() {
	var tm *town.Townmanager
	var gm *ghost.Ghostmanager

	timeout := time.Second * 0
	timeout2 := time.Second * 120
	for {
		select {
		case <-time.After(timeout):
			log.Info("town timeout should be running now")
			timeout = time.Minute * 45
			r.updateTime(0, timeout)
			err := r.checkTown()
			if err == nil {
				tm = nil
				tm = &town.Townmanager{User: r.Server.Config2.TownName, Password: r.Server.Config2.TownPassword, DB: r.Server.RelDB, Status: r.Server.StatusDB}
				go tm.Start()
			} else {
				timeout = time.Minute * 1
				r.updateTime(0, timeout)
				log.Info(err.Error())
				log.Info("town: trying login again in %v minute", timeout)
			}
		case <-time.After(timeout2):
			log.Info("ghost timeout should be running now")
			timeout2 = time.Minute * 49
			r.updateTime(1, timeout2)
			err := r.checkGhost()
			if err == nil {
				gm = nil
				gm = &ghost.Ghostmanager{User: r.Server.Config2.GhostName, Password: r.Server.Config2.GhostPassword, DB: r.Server.RelDB, Status: r.Server.StatusDB}
				go gm.Start()
			} else {
				timeout2 = time.Minute * 1
				r.updateTime(1, timeout2)
				log.Info(err.Error())
				log.Info("ghost: trying to login again in %v minute", timeout2)
			}

		}
	}
}

*/
