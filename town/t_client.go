package town

import (
	"errors"
	goquery "github.com/PuerkitoBio/goquery"
	log "github.com/dvirsky/go-pylog/logging"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Townclient struct {
	User, Password string
	cookies        []*http.Cookie
}

const (
	DUMP  = 0
	DAILY = "http://www.town.ag/v2/search.php?do=getnew"
	LOGIN = "http://www.town.ag/v2/login.php?do=login"
	ROOT  = "http://www.town.ag/v2/"
)

func (t *Townclient) getSValue() (sValue string) {
	log.Info("getting sValue for town login")
	sValue = ""
	var doc *goquery.Document
	var e error
	if doc, e = goquery.NewDocument(ROOT); e != nil {
		log.Error("couldn't connect to town")
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
	log.Info("sValue: %v", sValue)
	return sValue
}

// Logs into town.ag and returns the response cookies
func (t *Townclient) Login() error {
	log.Info("login process started")

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

	client := &http.Client{}
	req, err := http.NewRequest("POST", LOGIN, strings.NewReader(param.Encode()))
	if err != nil {
		return err
	}

	t.addHeader(req)

	t.dumpRequest(req, "town_login_req")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	t.dumpResponse(resp, "town_login_resp")
	t.cookies = resp.Cookies()
	return nil
}

func Redirect(req *http.Request, via []*http.Request) error {
	return errors.New("bla")
}

//return the Daily url or "" if something went wrong
func (t *Townclient) GetDailyUrl() (string, error) {
	log.Info("getting Daily Url for town")
	client := &http.Client{
		CheckRedirect: Redirect,
	}
	req, err := http.NewRequest("GET", DAILY, nil)
	if err != nil {
		log.Error(err.Error())
		return "", err
	}

	t.addHeader(req)

	if t.cookies != nil {
		for _, cookie := range t.cookies {
			req.AddCookie(cookie)
		}
	}

	resp, err := client.Do(req)
	if resp == nil {
		return "", err
	}

	time1 := time.Now()
	t.dumpResponse(resp, "daily"+strconv.Itoa(time1.Nanosecond()))

	url, err := resp.Location()
	if err != nil {
		return "", err
	}
	log.Info("daily url: %v", url.String())
	return url.String(), nil

}

//http get to the given ressource
func (t *Townclient) Get(sUrl string) (*http.Response, error) {
	log.Info("GET: %v", sUrl)
	client := &http.Client{}
	req, err := http.NewRequest("GET", sUrl, nil)
	if err != nil {
		log.Critical("couldn't create Request to: %v", sUrl)
		return nil, err
	}

	t.addHeader(req)

	if t.cookies != nil {
		for _, cookie := range t.cookies {
			req.AddCookie(cookie)
		}
	}

	time1 := time.Now()
	t.dumpRequest(req, "town_get_req_"+strconv.Itoa(time1.Nanosecond()))

	//connect to sUrl
	resp, err := client.Do(req)
	if err != nil {
		log.Error("couldn't connect to: %v", sUrl)
		return nil, err
	}

	t.dumpResponse(resp, "town_get_resp_"+strconv.Itoa(time1.Nanosecond()))

	return resp, nil
}

func (t *Townclient) dumpRequest(req *http.Request, name string) {
	if DUMP == 1 {
		dump1, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			log.Critical("dump of %v request failed", name)
		}
		ioutil.WriteFile(name, dump1, 0777)
	}
}

func (t *Townclient) dumpResponse(resp *http.Response, name string) {
	if DUMP == 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			log.Critical("log failed for get Request")
		}
		ioutil.WriteFile(name, dump, 0777)
	}
}

func (t *Townclient) addHeader(req *http.Request) {
	req.Header.Add("Host", "www.town.ag")
	//req.Header.Add("Origin", "www.town.ag")
	req.Header.Add("Referer", "http://www.town.ag/v2/")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/27.0.1453.116 Safari/537.36")
	req.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Add("Accept-Language", "de-DE,de;q=0.8,en-US;q=0.6,en;q=0.4")
	req.Header.Add("Cache-Control", "max-age=0")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Pragma", "no-cache")
	req.Header.Add("DNT", "1")
}
