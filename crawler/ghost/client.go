package ghost

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Kemonozume/nzbcrawler/util"
	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
)

const (
	DAILY   = "http://ghost-of-usenet.org/index.php/BoardQuickSearch/?mode=unreadPosts"
	LOGIN   = "http://ghost-of-usenet.org/index.php/Login/"
	TIMEOUT = time.Duration(2 * time.Second)
)

var client *http.Client = &http.Client{
	Timeout: TIMEOUT,
	/*Transport: &http.Transport{
		Proxy:              http.ProxyFromEnvironment,
		DisableKeepAlives:  true,
		DisableCompression: true,
	},/*/
}

var clientRed *http.Client = &http.Client{
	Timeout:       TIMEOUT,
	CheckRedirect: Redirect,
	/*Transport: &http.Transport{
		Proxy:              http.ProxyFromEnvironment,
		DisableKeepAlives:  true,
		DisableCompression: true,
	},/*/
}

func Redirect(req *http.Request, via []*http.Request) error {
	return errors.New("bla")
}

type GhostClient struct {
	User, Password string
	cookies        []*http.Cookie
	logged_in      bool
	dump           bool
}

func NewClient() (gc *GhostClient) {
	gc = &GhostClient{}
	gc.logged_in = false
	gc.dump = false
	return gc
}

func (g *GhostClient) SetAuth(user, password string) {
	g.User = user
	g.Password = password
}

func (g *GhostClient) SetDump(val bool) {
	g.dump = val
}

func (g GhostClient) IsLoggedIn() bool {
	return g.logged_in
}

func (g *GhostClient) getFirstTimeStuff() (tvalue string, err error) {
	log.WithField("tag", TAG).Infof("GET %v", LOGIN)
	log.WithField("tag", TAG).Info("getting cookies")

	req, err := http.NewRequest("GET", LOGIN, nil)
	if err != nil {
		return
	}

	req.Header.Add("Accept", "text/html, application/xhtml+xml, */*")
	req.Header.Add("Accept-Language", "de-DE")
	//req.Header.Add("Accept-Encoding", "gzip, deflate")
	req.Header.Add("User-Agent", "Mozilla/5.0 (compatible; MSIE 10.0; Windows NT 6.2; WOW64; Trident/6.0)")
	req.Header.Add("Host", "ghost-of-usenet.org")

	//connect to sUrl
	resp, err := client.Do(req)
	if err != nil {
		return
	}

	if g.dump {
		util.DumpRequest(req, "pre login request")
		util.DumpResponse(resp, "pre login response")
	}

	g.cookies = resp.Cookies()

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return
	}

	doc.Find("input").Each(func(i int, s *goquery.Selection) {
		attr, exists := s.Attr("name")
		if exists == true {
			if attr == "t" {
				bla, exists := s.Attr("value")
				if exists == true {
					tvalue = bla
				}
			}
		}

	})

	if tvalue == "" {
		return "", errors.New("no tvalue found")
	}

	resp.Close = true
	resp.Body.Close()

	return
}

// Logs into town.ag and returns the response cookies
func (g *GhostClient) Login() error {
	log.WithField("tag", TAG).Info("login process started")

	tvalue, err := g.getFirstTimeStuff()
	if err != nil {
		return err
	}

	param := url.Values{}

	param.Set("username", g.User)
	param.Add("action", "login")
	param.Add("password", g.Password)
	param.Add("useCookies", "1")
	param.Add("url", "")
	param.Add("t", tvalue)

	req, err := http.NewRequest("POST", LOGIN, strings.NewReader(param.Encode()))

	if err != nil {
		return err
	}

	log.WithField("tag", TAG).Infof("POST %v", LOGIN)

	if g.cookies != nil {
		for _, cookie := range g.cookies {
			req.AddCookie(cookie)
		}
	}
	req.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Add("Referer", "http://ghost-of-usenet.org/index.php/Login/")
	req.Header.Add("Accept-Language", "en-US,en;q=0.8,de;q=0.6")
	//req.Header.Add("Accept-Encoding", "gzip, deflate")
	req.Header.Add("Cache-Control", "max-age=0")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.3; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/40.0.2214.10 Safari/537.36")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Host", "ghost-of-usenet.org")
	req.Header.Add("Origin", "http://ghost-of-usenet.org")

	length := strconv.Itoa(len(param.Encode()))
	req.Header.Add("Content-Length", length)
	req.Header.Add("Pragma", "no-cache")

	if g.dump {
		util.DumpRequest(req, "login request")
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if g.dump {
		util.DumpResponse(resp, "login response")
	}

	fmt.Printf("\n%+v\n", g.cookies)
	tmpcookie := resp.Cookies()[0]

	for i, val := range g.cookies {
		if val.Name == "wcf_cookieHash" {
			g.cookies[i] = tmpcookie
			break
		}
	}

	fmt.Printf("\n%+v\n", g.cookies)
	g.logged_in = true
	resp.Close = true
	resp.Body.Close()
	return nil
}

//http get to the given ressource
func (g *GhostClient) Get(sUrl string) (*http.Response, error) {
	log.WithField("tag", TAG).Infof("GET %v", sUrl)

	req, err := http.NewRequest("GET", sUrl, nil)
	if err != nil {
		log.WithField("tag", TAG).Errorf("couldn't create Request to: %v", sUrl)
		return nil, err
	}

	req.Header.Add("Accept", "text/html, application/xhtml+xml, */*")
	req.Header.Add("Referer", "http://ghost-of-usenet.org/index.php")
	req.Header.Add("Accept-Language", "de-DE")
	req.Header.Add("Accept-Encoding", "gzip, deflate")
	req.Header.Add("User-Agent", "Mozilla/5.0 (compatible; MSIE 10.0; Windows NT 6.2; WOW64; Trident/6.0)")
	req.Header.Add("Host", "ghost-of-usenet.org")

	if g.cookies != nil {
		for _, cookie := range g.cookies {
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
func (g *GhostClient) GetDailyUrl() (string, error) {
	log.WithField("tag", TAG).Infof("GET url: %v", DAILY)
	req, err := http.NewRequest("GET", DAILY, nil)
	if err != nil {
		log.WithField("tag", TAG).Error(err.Error())
		return "", err
	}

	req.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Add("Referer", "http://ghost-of-usenet.org/")
	req.Header.Add("Accept-Language", "en-US,en;q=0.8,de;q=0.6")
	req.Header.Add("Accept-Encoding", "gzip, deflate, sdch")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.3; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/40.0.2214.10 Safari/537.36")
	req.Header.Add("Host", "ghost-of-usenet.org")
	req.Header.Add("Connection", "keep-alive")

	if g.cookies != nil {
		for _, cookie := range g.cookies {
			req.AddCookie(cookie)
		}
	}

	resp, err := clientRed.Do(req)
	if resp == nil {
		return "", err
	}

	if g.dump {
		util.DumpRequest(req, "daily request")
		util.DumpResponse(resp, "daily response")
	}

	resp.Close = true
	defer resp.Body.Close()

	url, err := resp.Location()
	if err != nil {
		return "", err
	}

	return url.String(), nil

}
