package runner

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/Kemonozume/nzbcrawler/config"
	"github.com/Kemonozume/nzbcrawler/crawler"
	"github.com/Kemonozume/nzbcrawler/crawler/town"
	"github.com/Kemonozume/nzbcrawler/data"
	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
)

type Runner struct {
	DB       *gorm.DB
	Timeout  time.Duration
	RecvChan chan []data.Release
	Config   *config.Config
	Manager  map[string]*crawler.Manager
}

const (
	TAG    = "runner"
	EXISTS = "SELECT id from releases WHERE checksum = ?"
)

func NewRunner(db *gorm.DB, conf *config.Config) (run *Runner, err error) {
	run = &Runner{}
	run.DB = db
	run.Config = conf
	run.Timeout, err = time.ParseDuration(conf.Timeout)
	if err != nil {
		return nil, err
	}

	run.RecvChan = make(chan []data.Release, 3)
	run.Manager = make(map[string]*crawler.Manager)
	return
}

func (r *Runner) initCrawler() {
	man := crawler.NewManager(r.Config.TownUser, r.Config.TownPassword, "town", r.RecvChan)
	man.SetClient(func() crawler.Client { return town.NewClient() })
	man.SetParser(func() crawler.Parser { return &town.TownParser{} })
	r.Manager["town"] = man

	/*man2 := crawler.NewManager(r.Config.GhostUser, r.Config.GhostPassword, "ghost", r.RecvChan)
	man2.SetClient(func() crawler.Client { return ghost.NewClient() })
	man2.SetParser(func() crawler.Parser { return &ghost.GhostParser{} })
	r.Manager["ghost"] = man2*/
}

func (r *Runner) Start(ex chan bool) {
	r.initCrawler()
	log.WithField("tag", TAG).Info("starting")
	for _, man := range r.Manager {
		go man.Start()
	}

	ticker := time.NewTicker(r.Timeout)
	log.WithField("tag", TAG).Infof("timeout is %s", r.Timeout)
	for {
		select {
		case <-ticker.C:
			log.WithField("tag", TAG).Info("tick crawler")
			for _, man := range r.Manager {
				go man.Start()
			}
		case releases := <-r.RecvChan:
			r.saveReleases(releases)
		case <-ex:
			log.WithField("tag", TAG).Info("closing")
			for _, man := range r.Manager {
				man.SetEnd(true)
			}
			ticker.Stop()
			return
		}
	}
}

func (r *Runner) saveReleases(releases []data.Release) {
	rel := releases[len(releases)-1]
	if r.EndPointFunction(releases[0]) {
		log.WithField("tag", TAG).Infof("%s %s found endpoint", rel.Name, releases[0].Name)
		r.Manager[rel.Name].SetEnd(true)
		return
	} else {
		releases = releases[0 : len(releases)-1]
		for _, release := range releases {
			for i, tag := range release.Tags {
				err := r.DB.Where("value = ?", tag.Value).Attrs("id", -1).Find(&tag).Error
				if err != nil {
					if err.Error() == "Record Not Found" {
						newtag := data.Tag{Value: tag.Value, Weight: 1}
						log.WithField("tag", TAG).Infof("creating new tag %v", newtag)
						r.DB.Create(&newtag)
						release.Tags[i] = newtag
					} else {
						log.WithField("tag", TAG).Error(err.Error())
					}

				} else {
					tag.Weight = tag.Weight + 1
					release.Tags[i] = tag
					r.DB.Raw("UPDATE tags SET weight = ? where id = ?", tag.Weight, tag.Id)
				}
			}
			err := r.DB.Create(&release).Error
			if release.Id == 0 && err != nil {
				switch err.(type) {
				case *mysql.MySQLError:
					//error 1062 is duplicate
					if err.(*mysql.MySQLError).Number != 1062 {
						log.WithField("tag", TAG).Error(err.Error())
					}

				default:
					if err.Error() != "UNIQUE constraint failed: releases.checksum" {
						log.WithField("tag", TAG).Errorf("failed to save release %+v", release)
						log.WithField("tag", TAG).Error(err.Error())
					}
				}
			} else {
				log.WithField("tag", TAG).Infof("saved %s", release.Name)
			}
		}
	}
}

func (r *Runner) EndPointFunction(rel data.Release) bool {
	rows, err := r.DB.Raw(EXISTS, rel.Checksum).Rows()
	if err != nil {
		return false
	}
	defer rows.Close()
	id := -1
	if rows.Next() {
		rows.Scan(&id)
		if id == -1 {
			return false
		} else {
			return true
		}
	} else {
		return false
	}
}
