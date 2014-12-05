package town

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	"strings"

	goquery "github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
)

const (
	DAILY   = "http://www.town.ag/v2/search.php?do=getnew"
	LOGIN   = "http://www.town.ag/v2/login.php?do=login"
	ROOT    = "http://www.town.ag/v2/"
	THANKS  = "http://www.town.ag/v2/ajax.php"
	TIMEOUT = time.Duration(2 * time.Second)
)

var client *http.Client = &http.Client{
	Timeout: TIMEOUT,
	Transport: &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		DisableKeepAlives: true,
	},
}

var clientRed *http.Client = &http.Client{
	Timeout:       TIMEOUT,
	CheckRedirect: Redirect,
	Transport: &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		DisableKeepAlives: true,
	},
}

func Redirect(req *http.Request, via []*http.Request) error {
	return errors.New("bla")
}

type TownClient struct {
	User, Password string
	cookies        []*http.Cookie
	logged_in      bool
	dump           bool
}

func NewClient() (tc *TownClient) {
	tc = &TownClient{}
	tc.logged_in = false
	tc.dump = false
	return tc
}

func (t *TownClient) SetAuth(user, password string) {
	t.User = user
	t.Password = password
}

func (t *TownClient) SetDump(val bool) {
	t.dump = val
}

func (t TownClient) IsLoggedIn() bool {
	return t.logged_in
}

func (t *TownClient) getSValue() (sValue string) {
	log.WithField("tag", TAG).Info("getting sValue for town login")
	sValue = ""
	var doc *goquery.Document
	var e error
	log.WithField("tag", TAG).Infof("GET %v", ROOT)
	if doc, e = goquery.NewDocument(ROOT); e != nil {
		log.WithField("tag", TAG).Errorf("%s", e.Error())
		return
	}

	doc.Find("input").Each(func(i int, s *goquery.Selection) {
		attr, exists := s.Attr("name")
		if exists == true {
			if attr == "s" {
				bla, exists := s.Attr("value")
				if exists == true {
					sValue = bla
				}
			}
		}

	})
	log.WithField("tag", TAG).Infof("sValue: %v", sValue)
	return sValue
}

// Logs into town.ag and returns the response cookies
func (t *TownClient) Login() error {
	log.WithField("tag", TAG).Info("login process started")

	sValue := t.getSValue()

	if sValue == "" {
		return errors.New("couldnt find SValue for the Town login")
	}

	param := url.Values{}
	param.Set("do", "login")
	param.Add("s", sValue)
	param.Add("securitytoken", "guest")
	param.Add("vb_login_username", t.User)
	param.Add("vb_login_password", "")
	param.Add("cookieuser", "1")
	param.Add("vb_login_md5password", t.Password)
	param.Add("vb_login_md5password_utf", t.Password)
	param.Add("url", "/v2/")

	req, err := http.NewRequest("POST", LOGIN, strings.NewReader(param.Encode()))
	if err != nil {
		return err
	}

	log.WithField("tag", TAG).Infof("POST %v", LOGIN)
	t.addHeader(req)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	t.cookies = resp.Cookies()

	t.logged_in = true
	resp.Close = true
	resp.Body.Close()
	return nil
}

//http get using the given sUrl
func (t *TownClient) Get(sUrl string) (*http.Response, error) {
	log.WithField("tag", TAG).Infof("GET %v", sUrl)

	req, err := http.NewRequest("GET", sUrl, nil)
	if err != nil {
		log.WithField("tag", TAG).Errorf("couldn't create Request to: %v", sUrl)
		return nil, err
	}

	t.addHeader(req)

	if t.cookies != nil {
		for _, cookie := range t.cookies {
			req.AddCookie(cookie)
		}
	}

	//connect to sUrl
	resp, err := client.Do(req)
	if err != nil {
		log.WithField("tag", TAG).Errorf("couldn't connect to: %v", sUrl)
		return nil, err
	}

	return resp, nil
}

//return the Daily url or "" if something went wrong
func (t *TownClient) GetDailyUrl() (string, error) {
	log.WithField("tag", TAG).Info("getting Daily Url for town")

	req, err := http.NewRequest("GET", DAILY, nil)
	if err != nil {
		log.WithField("tag", TAG).Error(err.Error())
		return "", err
	}

	t.addHeader(req)

	if t.cookies != nil {
		for _, cookie := range t.cookies {
			req.AddCookie(cookie)
		}
	}

	resp, err := clientRed.Do(req)
	if resp == nil {
		return "", err
	}

	resp.Close = true
	defer resp.Body.Close()

	lv := resp.Header.Get("Location")
	if lv == "" {
		return "", errors.New("no Location header|most likely town annoucment")
	}

	return lv, nil
}

//execute ajax thank request for a post
func (t *TownClient) ThankPost(postid string, token string) (err error) {
	log.WithField("tag", TAG).Infof("thanking post %s", postid)

	param := url.Values{}
	param.Set("do", "thanks")
	param.Add("postid", postid)
	param.Add("securitytoken", token)
	param.Add("s", "")

	req, err := http.NewRequest("POST", THANKS, strings.NewReader(param.Encode()))
	if err != nil {
		return
	}

	log.WithField("tag", TAG).Infof("POST url: %v", THANKS)
	t.addHeader(req)
	if t.cookies != nil {
		for _, cookie := range t.cookies {
			req.AddCookie(cookie)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return
	}

	resp.Close = true
	resp.Body.Close()
	return
}

func (t *TownClient) addHeader(req *http.Request) {
	req.Header.Add("Host", "www.town.ag")
	//req.Header.Add("Origin", "www.town.ag")
	req.Header.Add("Referer", "http://www.town.ag/v2/")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.3; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/36.0.1985.125 Safari/537.36")
	req.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Add("Accept-Language", "de-DE,de;q=0.8,en-US;q=0.6,en;q=0.4")
	req.Header.Add("Cache-Control", "max-age=0")
	req.Header.Add("Accept-Encoding", "gzip, deflate")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Pragma", "no-cache")
}
