package ghost

import (
	"errors"
	log "github.com/dvirsky/go-pylog/logging"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Ghostclient struct {
	User, Password string
	cookies        []*http.Cookie
}

const (
	DUMP  = 0
	DAILY = "http://ghost-of-usenet.org/search.php?action=24h"
	LOGIN = "http://ghost-of-usenet.org/login.php"
)

func (g *Ghostclient) getFirstTimeShit() error {
	log.Info("GET: Login Cookies")
	client := &http.Client{}
	req, err := http.NewRequest("GET", LOGIN, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Accept", "text/html, application/xhtml+xml, */*")
	req.Header.Add("Accept-Language", "de-DE")
	req.Header.Add("User-Agent", "Mozilla/5.0 (compatible; MSIE 10.0; Windows NT 6.2; WOW64; Trident/6.0)")
	req.Header.Add("Accept-Encoding", "gzip, deflate")
	req.Header.Add("Connection", "Keep-Alive")
	req.Header.Add("Host", "ghost-of-usenet.org")

	time1 := time.Now()
	g.dumpRequest(req, "ghost_first_req_"+strconv.Itoa(time1.Nanosecond()))

	//connect to sUrl
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	g.cookies = resp.Cookies()
	g.dumpResponse(resp, "ghost_first_resp_"+strconv.Itoa(time1.Nanosecond()))

	return nil
}

// Logs into town.ag and returns the response cookies
func (g *Ghostclient) Login() error {
	log.Info("login process started")

	g.getFirstTimeShit()

	param := url.Values{}
	param.Set("url", "index.php")
	param.Add("send", "send")
	param.Add("sid", "")
	param.Add("l_username", g.User)
	param.Add("l_password", g.Password)
	param.Add("submit", "Anmelden")

	client := &http.Client{}
	req, err := http.NewRequest("POST", LOGIN, strings.NewReader(param.Encode()))

	if err != nil {
		return err
	}

	if g.cookies != nil {
		for _, cookie := range g.cookies {
			req.AddCookie(cookie)
		}
	}
	req.Header.Add("Accept", "text/html, application/xhtml+xml, */*")
	req.Header.Add("Referer", "http://ghost-of-usenet.org/index.php")
	req.Header.Add("Accept-Language", "de-DE")
	req.Header.Add("User-Agent", "Mozilla/5.0 (compatible; MSIE 10.0; Windows NT 6.2; WOW64; Trident/6.0)")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept-Encoding", "gzip, deflate")
	req.Header.Add("Host", "ghost-of-usenet.org")

	length := strconv.Itoa(len(param.Encode()))
	req.Header.Add("Content-Length", length)
	req.Header.Add("Connection", "Keep-Alive")
	req.Header.Add("Pragma", "no-cache")

	g.dumpRequest(req, "town_login_req")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	g.dumpResponse(resp, "town_login_resp")
	g.cookies = resp.Cookies()
	return nil
}

//http get to the given ressource
func (g *Ghostclient) Get(sUrl string) (*http.Response, error) {
	log.Info("GET: %v", sUrl)
	client := &http.Client{}
	req, err := http.NewRequest("GET", sUrl, nil)
	if err != nil {
		log.Critical("couldn't create Request to: %v", sUrl)
		return nil, err
	}

	req.Header.Add("Accept", "text/html, application/xhtml+xml, */*")
	req.Header.Add("Referer", "http://ghost-of-usenet.org/index.php")
	req.Header.Add("Accept-Language", "de-DE")
	req.Header.Add("User-Agent", "Mozilla/5.0 (compatible; MSIE 10.0; Windows NT 6.2; WOW64; Trident/6.0)")
	req.Header.Add("Accept-Encoding", "gzip, deflate")
	req.Header.Add("Host", "ghost-of-usenet.org")
	req.Header.Add("Connection", "Keep-Alive")

	if g.cookies != nil {
		for _, cookie := range g.cookies {
			req.AddCookie(cookie)
		}
	}

	time1 := time.Now()
	g.dumpRequest(req, "ghost_get_req_"+strconv.Itoa(time1.Nanosecond()))

	//connect to sUrl
	resp, err := client.Do(req)
	if err != nil {
		log.Error("couldn't connect to: %v", sUrl)
		return nil, err
	}

	g.dumpResponse(resp, "ghost_get_resp_"+strconv.Itoa(time1.Nanosecond()))

	return resp, nil
}

func Redirect(req *http.Request, via []*http.Request) error {
	return errors.New("bla")
}

//return the Daily url or "" if something went wrong
func (g *Ghostclient) GetDailyUrl() (string, error) {
	log.Info("getting Daily Url for ghost")
	client := &http.Client{
		CheckRedirect: Redirect,
	}
	req, err := http.NewRequest("GET", DAILY, nil)
	if err != nil {
		log.Error(err.Error())
		return "", err
	}

	req.Header.Add("Accept", "text/html, application/xhtml+xml, */*")
	req.Header.Add("Referer", "http://ghost-of-usenet.org/index.php")
	req.Header.Add("Accept-Language", "de-DE")
	req.Header.Add("User-Agent", "Mozilla/5.0 (compatible; MSIE 10.0; Windows NT 6.2; WOW64; Trident/6.0)")
	req.Header.Add("Accept-Encoding", "gzip, deflate")
	req.Header.Add("Host", "ghost-of-usenet.org")
	req.Header.Add("Connection", "Keep-Alive")

	if g.cookies != nil {
		for _, cookie := range g.cookies {
			req.AddCookie(cookie)
		}
	}

	time1 := time.Now()
	g.dumpRequest(req, "daily_req_"+strconv.Itoa(time1.Nanosecond()))

	resp, err := client.Do(req)
	if resp == nil {
		return "", err
	}

	g.dumpResponse(resp, "daily_resp_"+strconv.Itoa(time1.Nanosecond()))

	url, err := resp.Location()
	if err != nil {
		return "", err
	}
	log.Info("daily url: %v", url.String())
	return url.String(), nil

}

func (g *Ghostclient) dumpRequest(req *http.Request, name string) {
	if DUMP == 1 {
		dump1, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			log.Critical("dump of %v request failed", name)
		}
		ioutil.WriteFile(name, dump1, 0777)
	}
}

func (g *Ghostclient) dumpResponse(resp *http.Response, name string) {
	if DUMP == 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			log.Critical("log failed for get Request")
		}
		ioutil.WriteFile(name, dump, 0777)
	}
}
