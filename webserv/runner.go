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
	go func() {
		timeout := time.Second * 1
		for {
			select {
			case <-time.After(timeout):
				if !r.checkTown() {
					timeout = time.Minute * 1
					log.Info("town: trying login again in %v minute", timeout)
				} else {
					tm := town.Townmanager{User: r.Server.Config2.TownName, Password: r.Server.Config2.TownPassword, DB: r.Server.RelDB}
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
					gm := ghost.Ghostmanager{User: r.Server.Config2.GhostName, Password: r.Server.Config2.GhostPassword, DB: r.Server.RelDB}
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
