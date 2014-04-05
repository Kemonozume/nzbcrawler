package webserv

import (
	"./../ghost"
	"./../town"
	log "github.com/dvirsky/go-pylog/logging"
	"time"
)

type Runner struct {
	Server *Server
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

func (r *Runner) Start() {
	go r.StartTown()
	go r.StartGhost()
}

func (r *Runner) StartTown() {
	defer func() {
		if a := recover(); a != nil {
			log.Error("Recovered TownCrawler crash %v", a)
			r.StartTown()
		}
	}()
	timeout := time.Second * 1
	for {
		select {
		case <-time.After(timeout):
			if !r.checkTown() {
				timeout = time.Minute * 5
				log.Info("town: trying login again in %v minute", timeout)
			} else {
				tm := town.Townmanager{User: r.Server.Config2.TownName, Password: r.Server.Config2.TownPassword, DB: r.Server.RelDB}
				go tm.Start()
				dur, err := time.ParseDuration(r.Server.Config2.Timeout)
				if err != nil {
					log.Error(err.Error())
				}
				c := time.Tick(dur)
				for _ = range c {
					log.Info("tick start town")
					go tm.Start()
				}
			}
		}
		break
	}
	return
}

func (r *Runner) StartGhost() {
	defer func() {
		if a := recover(); a != nil {
			log.Error("Recovered GhostCrawler crash %v", a)
			r.StartGhost()
		}
	}()
	timeout := time.Second * 2
	for {
		select {
		case <-time.After(timeout):
			if !r.checkTown() {
				timeout = time.Minute * 5
				log.Info("ghost: trying to login again in %v minute", timeout)
			} else {
				gm := ghost.Ghostmanager{User: r.Server.Config2.GhostName, Password: r.Server.Config2.GhostPassword, DB: r.Server.RelDB}
				go gm.Start()
				dur, err := time.ParseDuration(r.Server.Config2.Timeout)
				if err != nil {
					log.Error(err.Error())
				}
				c := time.Tick(dur)
				for _ = range c {
					log.Info("tick start ghost")
					go gm.Start()
				}
			}
		}
		break
	}
	return
}
