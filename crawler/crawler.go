package crawler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Kemonozume/nzbcrawler/data"
	log "github.com/sirupsen/logrus"
)

const TAG = "crawler"

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
	name           string
	client         Client
	parser         Parser
	client_func    func() Client
	parser_func    func() Parser
}

func NewManager(user, password, name string, ch chan []data.Release) (m *Manager) {
	m = &Manager{User: user, Password: password, send_chan: ch, name: name}
	return m
}

func (m *Manager) SetEnd(end bool) {
	m.end = end
}

func (m *Manager) SetParser(s func() Parser) {
	m.parser_func = s
}

func (m *Manager) SetClient(s func() Client) {
	m.client_func = s
}

func (m *Manager) Start() (err error) {
	log.WithField("tag", TAG).Info("Manager starting")
	m.end = false

	m.client = m.client_func()
	m.parser = m.parser_func()

	m.client.SetAuth(m.User, m.Password)

	err = m.client.Login()
	if err != nil {
		log.WithField("tag", TAG).Errorf("%s login failed", m.name)
		return
	} else {
		log.WithField("tag", TAG).Infof("%s login successful", m.name)
	}

	url, err := m.client.GetDailyUrl()
	if err != nil {
		log.WithField("tag", TAG).Errorf("%s daily url not found", m.name)
		return
	} else {
		log.WithField("tag", TAG).Infof("%s daily url: %s", m.name, url)
	}

	err = m.parser.ParseUrlWithClient(url, &m.client)
	if err != nil {
		log.WithField("tag", TAG).Errorf("%s couldnt parse html body", m.name)
		return
	}

	m.maxpage = m.parser.GetMaxPage()
	if m.maxpage == -1 {
		log.WithField("tag", TAG).Errorf("%s pagecount failed", m.name)
		return errors.New("pagecount failed")
	}

	tmp := m.parser.ParseReleases()
	if len(tmp) > 0 {
		tmp = append(tmp, data.Release{Name: m.name})
		m.send_chan <- tmp
	}

	i := 2

	for {
		m.parser = m.parser_func()
		err = m.parser.ParseUrlWithClient(url+"&pp=25&page="+strconv.Itoa(i), &m.client)
		if err != nil {
			log.WithField("tag", TAG).Error(err.Error())
			time.Sleep(2 * time.Second)
			i++
			continue
		}

		rel := m.parser.ParseReleases()
		if len(rel) > 0 {
			rel = append(rel, data.Release{Name: m.name})
			m.send_chan <- rel
		}

		log.WithField("tag", TAG).Infof("%s crawled page %d/%d", m.name, i, m.maxpage)
		time.Sleep(2 * time.Second)

		i++
		if i == m.maxpage+1 {
			break
		}

		if m.end {
			log.WithField("tag", TAG).Infof("%s shutting down", m.name)
			break
		}

	}
	log.WithField("tag", TAG).Infof("%s parser finished", m.name)
	return
}
