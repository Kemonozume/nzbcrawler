package town

import (
	"./../mydb"
	log "github.com/dvirsky/go-pylog/logging"
	"strconv"
	"time"
)

type Townmanager struct {
	User, Password, url string
	DB                  *mydb.MyDB
	page, maxpage       int
	end                 bool
}

func (t *Townmanager) Start() {
	log.Info("townManager start")

	t.page = 1
	t.end = false

	tc := &Townclient{}
	tc.User = t.User
	tc.Password = t.Password

	err := t.init(tc)
	log.Info("townclient init finished, starting to parse...")
	if err != nil {
		log.Error("Townclient init failed")
		log.Error(err.Error())
		return
	}

	tp := &Townparser{Url: t.url, Tc: tc}

	count, err := tp.ParsePageCount()
	if err != nil {
		log.Error(err.Error())
		return
	}
	t.maxpage = count
	log.Info("townclient crawling approximately %v pages", t.maxpage)
	t.saveReleases(tp.Rel)

	i := 1
	for {
		if i == 1 {
			err = tp.ParseReleases(false)
			if err != nil {
				log.Error(err.Error())
				break
			}
		} else {
			tp = nil
			tp = &Townparser{Url: t.url + "&pp=25&page=" + strconv.Itoa(i), Tc: tc}
			err = tp.ParseReleases(true)
			if err != nil {
				log.Error(err.Error())
				break
			}
		}
		log.Info("town: crawled page %v/%v", i, t.maxpage)
		t.saveReleases(tp.Rel)
		time.Sleep(5 * time.Second)
		i++
		if i == t.maxpage+1 {
			break
		}
		if t.end {
			log.Info("found old end point")
			break
		}
	}
	log.Info("town parser closing")

}

func (t *Townmanager) saveReleases(releases []Release) {
	log.Info("saving %d releases", len(releases))
	for _, rel := range releases {
		_, err := t.DB.Eng.Exec("INSERT INTO release VALUES(?, ?, ?, ?, ?, ?, ?)", rel.Checksum, rel.Url, rel.Name, rel.Tag, rel.Time, rel.Hits, rel.Rating)
		if err != nil {
			log.Error(err.Error())
			t.end = true
			break
		}
	}

}

func (t *Townmanager) init(tc *Townclient) error {

	if err := t.DB.Eng.CreateTables(&Release{}); err != nil {
		log.Error(err.Error())
	}

	//login to get cookies
	err := tc.Login()
	if err != nil {
		log.Error(err.Error())
		return err
	}

	time.Sleep(1000 * time.Millisecond)

	url, err := tc.GetDailyUrl()
	if err != nil {
		log.Error(err.Error())
		return err
	}

	t.url = url

	time.Sleep(200 * time.Millisecond)

	return nil
}
