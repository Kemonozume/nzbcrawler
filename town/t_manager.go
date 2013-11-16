package town

import (
	"code.google.com/p/go-sqlite/go1/sqlite3"
	"github.com/coopernurse/gorp"
	log "github.com/dvirsky/go-pylog/logging"
	"strconv"
	"time"
)

type Townmanager struct {
	User, Password, url string
	DB                  *gorp.DbMap
	page, maxpage       int
	end                 bool
}

const (
	TAG = "[town]"
)

func (t *Townmanager) Start() {
	t.page = 1
	t.end = false

	tc := &Townclient{}
	tc.User = t.User
	tc.Password = t.Password

	err := t.init(tc)
	log.Info("%s init finished, starting to parse...", TAG)
	if err != nil {
		log.Error("%s init failed", TAG)
		log.Error("%s %s", TAG, err.Error())
		return
	}

	tp := &Townparser{Url: t.url, Tc: tc}

	count, err := tp.ParsePageCount()
	if err != nil {
		log.Error("%s %s", TAG, err.Error())
		return
	}
	t.maxpage = count
	log.Info("%s crawling approximately %v pages", TAG, t.maxpage)
	t.saveReleases(tp.Rel)

	i := 1
	for {
		if i == 1 {
			err = tp.ParseReleases(false)
			if err != nil {
				log.Error("%s %s", TAG, err.Error())
				break
			}
		} else {
			tp = nil
			tp = &Townparser{Url: t.url + "&pp=25&page=" + strconv.Itoa(i), Tc: tc}
			err = tp.ParseReleases(true)
			if err != nil {
				log.Error("%s %s", TAG, err.Error())
				break
			}
		}
		log.Info("%s crawled page %v/%v", TAG, i, t.maxpage)
		t.saveReleases(tp.Rel)
		time.Sleep(5 * time.Second)
		i++
		if i == t.maxpage+1 {
			break
		}
		if t.end {
			log.Info("%s found old end point", TAG)
			break
		}
	}
	log.Info("%s parser closing", TAG)

}

func (t *Townmanager) saveReleases(releases []Release) {
	for _, rel := range releases {
		err := t.DB.Insert(&rel)
		if err != nil {
			switch err.(type) {
			case *sqlite3.Error:
				if err.(*sqlite3.Error).Code() == 2067 {
					t.end = true
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

func (t *Townmanager) init(tc *Townclient) error {
	//login to get cookies
	err := tc.Login()
	if err != nil {
		log.Error("%s %s", TAG, err.Error())
		return err
	}

	time.Sleep(1000 * time.Millisecond)

	url, err := tc.GetDailyUrl()
	if err != nil {
		log.Error("%s %s", TAG, err.Error())
		return err
	}

	t.url = url

	time.Sleep(200 * time.Millisecond)

	return nil
}
