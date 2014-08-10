package crawler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Kemonozume/nzbcrawler/data"
	log "github.com/sirupsen/logrus"
)

const TAG = "[crawler]"

type Parser interface {
	ParseUrlWithClient(url string, client *Client) error
	//return -1 if no page count was found
	GetMaxPage() int
	ParseReleases() []data.Release
}

type Client interface {
	SetAuth(user, password string)
	IsLoggedIn() bool
	Login() error
	Get(url string) (*http.Response, error)
	GetDailyUrl() (string, error)
}

type Manager struct {
	User, Password string
	send_chan      chan []data.Release
	maxpage        int
	end            bool
	end_check      func(rel data.Release) bool
	parser         Parser
	client         Client
	name           string
}

func NewManager(user, password, name string, ch chan []data.Release) (m *Manager) {
	m = &Manager{User: user, Password: password, send_chan: ch, name: name}
	return m
}

func (m *Manager) SetEnd(end bool) {
	m.end = end
}

func (m *Manager) SetEndPointFunction(s func(rel data.Release) bool) {
	m.end_check = s
}

func (m *Manager) SetParser(par Parser) {
	m.parser = par
}

func (m *Manager) SetClient(client Client) {
	m.client = client
}

func (m *Manager) Start() (err error) {
	log.Infof("%s Manager starting", TAG)
	m.end = false

	m.client.SetAuth(m.User, m.Password)

	err = m.client.Login()
	if err != nil {
		log.Errorf("%s %s login failed", TAG, m.name)
		return
	} else {
		log.Infof("%s %s login successful", TAG, m.name)
	}

	url, err := m.client.GetDailyUrl()
	if err != nil {
		log.Errorf("%s %s daily url not found", TAG, m.name)
		return
	} else {
		log.Infof("%s %s daily url: %s", TAG, m.name, url)
	}

	err = m.parser.ParseUrlWithClient(url, &m.client)
	if err != nil {
		log.Errorf("%s %s couldnt parse html body", TAG, m.name)
		return
	}

	m.maxpage = m.parser.GetMaxPage()
	if m.maxpage == -1 {
		log.Errorf("%s %s pagecount failed", TAG, m.name)
		return errors.New("pagecount failed")
	}

	i := 1

	for {
		if i != 1 {
			err = m.parser.ParseUrlWithClient(url+"&pp=25&page="+strconv.Itoa(i), &m.client)
			if err != nil {
				log.Errorf("%s %s", TAG, err.Error())
				time.Sleep(2 * time.Second)
				i++
				continue
			}
		}

		rel := m.parser.ParseReleases()
		if len(rel) > 0 {
			if m.end_check(rel[0]) {
				log.Infof("%s %s found old endpoint, closing", TAG, m.name)
				break
			}
			m.send_chan <- rel
		}

		log.Infof("%s %s crawled page %d/%d", TAG, m.name, i, m.maxpage)
		time.Sleep(2 * time.Second)

		i++
		if i == m.maxpage+1 {
			break
		}

		if m.end {
			log.Infof("%s %s shutting down", TAG, m.name)
			break
		}

	}
	log.Infof("%s %s parser finished", TAG, m.name)
	return
}
