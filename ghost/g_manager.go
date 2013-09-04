package ghost

import (
	"./../mydb"
	log "github.com/dvirsky/go-pylog/logging"
	"strconv"
	_ "strings"
	"time"
)

type Ghostmanager struct {
	User, Password, url string
	DB                  *mydb.MyDB
	Status              *mydb.MyDB
	maxpage             int
	end                 bool
}

func (g *Ghostmanager) Start() {

	g.Status.Mutex.Lock()
	g.Status.Eng.Exec("update status_runner set Running=? where id=?", true, 1)
	g.Status.Mutex.Unlock()

	g.end = false

	gc := &Ghostclient{}
	gc.User = g.User
	gc.Password = g.Password

	err := g.init(gc)
	log.Info("Ghostclient init finished, starting to parse...")
	if err != nil {
		log.Error("Ghostclient init failed")
		log.Error(err.Error())
		return
	}

	tp := &Ghostparser{Url: g.url, Gc: gc}

	i := 1
	for {
		if i == 1 {
			err = tp.ParseReleases()
			if err != nil {
				log.Error(err.Error())
				break
			}
			g.maxpage = tp.Count
			log.Info("Ghostclient crawling approximately %v pages", g.maxpage)
		} else {
			tp = nil
			tp = &Ghostparser{Url: g.url + "&page=" + strconv.Itoa(i), Gc: gc}
			err = tp.ParseReleases()
			if err != nil {
				log.Error(err.Error())
				break
			}
		}
		g.saveReleases(tp.Rel)
		log.Info("ghost: crawled page %v/%v", i, g.maxpage)
		time.Sleep(5 * time.Second)
		i++
		if i == g.maxpage+1 {
			break
		}
		if g.end {
			log.Info("ghost parser closing, found old end point")
			break
		}
	}

	g.Status.Mutex.Lock()
	g.Status.Eng.Exec("update status_runner set Running=? where id=?", false, 1)
	g.Status.Mutex.Unlock()
}

func (g *Ghostmanager) saveReleases(releases []Release) {

	for _, rel := range releases {
		g.DB.Mutex.Lock()
		id, err := g.DB.Eng.Insert(rel)
		g.DB.Mutex.Unlock()
		if err != nil && id == -1 {
			g.end = true
			break
		}
		if err != nil {
			log.Error(err.Error())
		}
	}

}

func (g *Ghostmanager) init(gc *Ghostclient) error {

	//create database tables
	g.DB.Mutex.Lock()

	if err := g.DB.Eng.CreateTables(&Release{}); err != nil {
		log.Error(err.Error())
	}

	g.DB.Mutex.Unlock()

	//login to get cookies
	err := gc.Login()
	if err != nil {
		log.Error(err.Error())
		return err
	}

	time.Sleep(1000 * time.Millisecond)

	url, err := gc.GetDailyUrl()
	if err != nil {
		log.Error(err.Error())
		return err
	}

	g.url = url

	time.Sleep(200 * time.Millisecond)

	return nil
}
