package ghost

import (
	"strconv"
	"time"
	"./../data"
	"code.google.com/p/go-sqlite/go1/sqlite3"
	"github.com/coopernurse/gorp"
	log "github.com/dvirsky/go-pylog/logging"
)

type Ghostmanager struct {
	User, Password, url string
	DB                  *gorp.DbMap
	maxpage             int
	end                 bool
}

const (
	TAG = "[ghost]"
)

func (g *Ghostmanager) Start() {

	g.end = false

	gc := &Ghostclient{}
	gc.User = g.User
	gc.Password = g.Password

	err := g.init(gc)
	log.Info("%s init finished, starting to parse...", TAG)
	if err != nil {
		log.Error("%s init failed", TAG)
		log.Error("%s %s", TAG, err.Error())
		return
	}

	tp := &Ghostparser{Url: g.url, Gc: gc}

	i := 1
	for {
		if i == 1 {
			err = tp.ParseReleases()
			if err != nil {
				log.Error("%s %s", TAG, err.Error())
				break
			}
			g.maxpage = tp.Count
			log.Info("%s crawling approximately %v pages", TAG, g.maxpage)
		} else {
			tp = nil
			tp = &Ghostparser{Url: g.url + "&page=" + strconv.Itoa(i), Gc: gc}
			err = tp.ParseReleases()
			if err != nil {
				log.Error("%s %s", TAG, err.Error())
				break
			}
		}
		g.saveReleases(tp.Rel)
		log.Info("%s crawled page %v/%v", TAG, i, g.maxpage)
		time.Sleep(5 * time.Second)
		i++
		if i == g.maxpage+1 {
			break
		}
		if g.end {
			log.Info("%s found old end point", TAG)
			break
		}
	}
	log.Info("%s closing", TAG)
}

func (g *Ghostmanager) saveReleases(releases []data.Release) {
	for _, rel := range releases {
		err := g.DB.Insert(&data.Release{Name: rel.Name, Checksum: rel.Checksum, Rating: rel.Rating, Hits: rel.Hits, Time: rel.Time, Url: rel.Url, Tag: rel.Tag})
		if err != nil {
			switch err.(type) {
			case *sqlite3.Error:
				if err.(*sqlite3.Error).Code() == 2067 {
					g.end = true
					break
				} else {
					log.Error("%s %s", TAG, err.Error())
				}
			default:
				log.Error("%s %s", TAG, err.Error())
			}
		} else {
			log.Info("%s saved %v", TAG, rel.Name)
		}
	}

}

func (g *Ghostmanager) init(gc *Ghostclient) error {

	//login to get cookies
	err := gc.Login()
	if err != nil {
		log.Error("%s %s", TAG, err.Error())
		return err
	}

	time.Sleep(1000 * time.Millisecond)

	url, err := gc.GetDailyUrl()
	if err != nil {
		log.Error("%s %s", TAG, err.Error())
		return err
	}

	g.url = url

	time.Sleep(200 * time.Millisecond)

	return nil
}
