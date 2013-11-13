package ghost

import (
	"../town"
	"github.com/coopernurse/gorp"
	log "github.com/dvirsky/go-pylog/logging"
	"strconv"
	"time"
)

type Ghostmanager struct {
	User, Password, url string
	DB                  *gorp.DbMap
	maxpage             int
	end                 bool
}

func (g *Ghostmanager) Start() {

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
			log.Info("found old end point")
			break
		}
	}
	log.Info("ghost parser closing")
}

func (g *Ghostmanager) saveReleases(releases []Release) {
	for _, rel := range releases {
		err := g.DB.Insert(&town.Release{Name: rel.Name, Checksum: rel.Checksum, Rating: rel.Rating, Hits: rel.Hits, Time: rel.Time, Url: rel.Url, Tag: rel.Tag})
		if err != nil {
			log.Error(err.Error())
			g.end = true
			break
		} else {
			log.Info("saved %v", rel.Name)
		}
	}

}

func (g *Ghostmanager) init(gc *Ghostclient) error {

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
