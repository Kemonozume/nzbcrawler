package ghost

import (
	"bytes"
	"io/ioutil"
	"time"

	"github.com/Kemonozume/nzbcrawler/crawler"
	"github.com/Kemonozume/nzbcrawler/data"
	"github.com/Kemonozume/nzbcrawler/util"
	"github.com/PuerkitoBio/goquery"

	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"golang.org/x/net/html"
)

type GhostParser struct {
	doc    *goquery.Document
	Rel    []data.Release
	client *crawler.Client
}

func (g *GhostParser) ParseUrlWithClient(url string, client *crawler.Client) (err error) {
	g.client = client
	cl := *client
	if !cl.IsLoggedIn() {
		err = cl.Login()
		if err != nil {
			return
		}
	}

	resp, err := cl.Get(url)
	if err != nil {
		return
	}

	defer resp.Body.Close()
	resp.Close = true

	bv, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	bv = util.BytesToUtf8(bv)

	g.doc, err = goquery.NewDocumentFromReader(bytes.NewReader(bv))
	return
}

func (g *GhostParser) getUrlAndTagAndName(rel *data.Release, sc *goquery.Selection) {
	sc.Find("a").Each(func(d int, sd *goquery.Selection) {
		switch d {
		case 0:
			if attr, exist := sd.Attr("href"); exist {
				rel.Name = sd.Text()
				rel.Url = g.clearUrl("http://ghost-of-usenet.org/" + attr)
			}
		case 1:
			if attr, exist := sd.Attr("href"); exist {
				if rel.Name != "" {
					text := sd.Text()
					text = strings.Replace(text, "/", ",", -1)
					text = strings.Replace(text, "\\", ",", -1)
					text = strings.Replace(text, "&", ",", -1)
					text = strings.Replace(text, "(", ",", -1)
					text = strings.Replace(text, ")", ",", -1)
					if strings.Contains(text, ",") {
						atext := strings.Split(text, ",")
						for _, tag := range atext {
							rel.AddTag(tag)
						}
					} else {
						rel.AddTag(text)
					}
					i := g.getBoardId(attr)
					if i != -1 {
						g.checkCat(rel, i)
					} else {
						rel.Name = ""
						rel.Checksum = ""
					}
				}
			}
		}
	})
}

func (g *GhostParser) getImageUrl(url string) (url2 string) {
	cl := *g.client
	resp, err := cl.Get(url)
	if err != nil {
		log.WithField("tag", TAG).Error(err.Error())
		return url2
	}
	defer resp.Body.Close()
	resp.Close = true

	respbody, err := html.Parse(resp.Body)
	doc := goquery.NewDocumentFromNode(respbody)

	doc.Find(".resizeImage").Each(func(a int, sa *goquery.Selection) {
		if a == 0 {
			if attr, exist := sa.Attr("src"); exist {
				url2 = attr
			}
		}
	})

	time.Sleep(2 * time.Second)

	return url2
}

func (g *GhostParser) clearUrl(url string) string {
	if strings.Contains(url, "sid") {
		astr := strings.Split(url, "&")
		return astr[0]
	}
	return url
}

func (g *GhostParser) getBoardId(str string) int {
	regex, err := regexp.Compile("boardid=.+&")
	if err != nil {
		log.WithField("tag", TAG).Error(err.Error())
		return -1
	}

	str = regex.FindString(str)
	str = strings.Replace(str, "&", "", -1)
	astr := strings.Split(str, "=")
	if len(astr) < 2 {
		return -1
	}
	i, err := strconv.Atoi(astr[1])
	if err != nil {
		log.WithField("tag", TAG).Errorf(err.Error())
		return -1
	}
	return i
}

func (g *GhostParser) GetMaxPage() (pagec int) {
	pagec = -1
	sel := g.doc.Find("table").Eq(51).Find("a").Eq(3)
	tmp, err := strconv.Atoi(sel.Text())
	if err != nil {
		return
	}
	pagec = tmp
	return
}

//parse the http resp from Townclient
func (g *GhostParser) ParseReleases() []data.Release {
	rel := data.Release{}
	se := g.doc.Find("table").Eq(10)

	se.Find("tr").Each(func(b int, sb *goquery.Selection) {
		sb.Find("td").Each(func(c int, sc *goquery.Selection) {
			if c == 2 {
				rel = data.Release{}
				g.getUrlAndTagAndName(&rel, sc)
				if rel.Name != "" {
					rel.Image = g.getImageUrl(rel.Url)
					rel.Time = time.Now().Unix()
					rel.EncodeName()
					g.checkQual(&rel)
					if rel.Name != "" {
						rel.Hits = 0
						g.Rel = append(g.Rel, rel)
					}
				}
			}
		})
	})

	return g.Rel
}

func (g *GhostParser) checkQual(r *data.Release) {
	if r.HasTag("music") || r.HasTag("games") {
		return
	}
	if strings.Contains(r.Name, "1080") {
		r.AddTag("1080")
	} else if strings.Contains(r.Name, "720") {
		r.AddTag("720")
	} else if strings.Contains(r.Name, "untouched") {
		r.AddTag("untouched")
	} else if strings.Contains(r.Name, "3d") {
		r.AddTag("3d")
	} else if !strings.Contains(r.Url, "720") && !strings.Contains(r.Url, "1080") && !strings.Contains(r.Url, "untouched") && !strings.Contains(r.Url, "3d") {
		r.AddTag("sd")
	}
}

func (g *GhostParser) checkCat(r *data.Release, boardid int) {
	switch boardid {
	case 26:
		r.AddTag("cinema")
	case 176:
		r.AddTag("cinema")
		r.AddTag("hd")
	case 28, 29, 59:
		r.AddTag("cinema")
		r.AddTag("sd")
	case 101:
		r.AddTag("movies")
	case 124, 125, 127, 1, 4, 6, 7:
		r.AddTag("movies")
		r.AddTag("sd")
	case 157, 158, 159, 211, 212, 160, 119, 120, 121, 213, 214, 122, 150, 202, 203, 141:
		r.AddTag("movies")
		r.AddTag("hd")
	case 143, 144, 145, 146:
		r.AddTag("movies")
		r.AddTag("hd")
		r.AddTag("eng")
	case 53:
		r.AddTag("series")
	case 77, 65, 215:
		r.AddTag("series")
		r.AddTag("sd")
	case 216, 217:
		r.AddTag("series")
		r.AddTag("hd")
	case 175, 208:
		r.AddTag("series")
		r.AddTag("hd")
		r.AddTag("eng")
	case 225:
		r.AddTag("series")
		r.AddTag("sd")
		r.AddTag("eng")
	case 12, 21, 19:
		r.AddTag("music")
	case 166:
		r.AddTag("music")
		r.AddTag("rock")
	case 167:
		r.AddTag("music")
		r.AddTag("pop")
	case 168:
		r.AddTag("music")
		r.AddTag("jazz")
		r.AddTag("blues")
		r.AddTag("souls")
		r.AddTag("country")
		r.AddTag("reggae")
	case 169:
		r.AddTag("music")
		r.AddTag("hip-hop")
	case 170:
		r.AddTag("music")
		r.AddTag("electronic")
	case 171:
		r.AddTag("music")
		r.AddTag("schlager")
		r.AddTag("volksmusik")
	case 172:
		r.AddTag("music")
		r.AddTag("oldies")
	case 173:
		r.AddTag("music")
		r.AddTag("metal")
	case 174:
		r.AddTag("music")
		r.AddTag("soundtrack")
	case 48:
		r.AddTag("music")
		r.AddTag("hörbuch")
	case 147:
		r.AddTag("music")
		r.AddTag("classic")
	case 177:
		r.AddTag("music")
		r.AddTag("discography")
	case 51:
		r.AddTag("games")
		r.AddTag("pc")
	case 32:
		r.AddTag("games")
		r.AddTag("xbox360")
	case 34:
		r.AddTag("games")
		r.AddTag("ps")
	case 37:
		r.AddTag("games")
		r.AddTag("wii")
	case 74:
		r.AddTag("xxx")
	case 85:
		r.AddTag("xxx")
		r.AddTag("hd")
	case 70, 71, 133:
		r.AddTag("xxx")
		r.AddTag("sd")
	default:
	}
}
