package ghost

import (
	"code.google.com/p/go.net/html"
	"crypto/sha1"
	"encoding/hex"
	"github.com/PuerkitoBio/goquery"
	log "github.com/dvirsky/go-pylog/logging"
	"github.com/nfnt/resize"
	"image/jpeg"
	"image/png"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Release struct {
	Checksum string `xorm:"pk not null"`
	Url      string
	Name     string
	Tag      string
	Time     int64
	Image    string `xorm:"-"`
}

type Ghostparser struct {
	Url   string
	Gc    *Ghostclient
	Rel   []Release
	Count int
}

func toUtf8(iso8859_1_buf []byte) string {
	buf := make([]rune, len(iso8859_1_buf))
	for i, b := range iso8859_1_buf {
		buf[i] = rune(b)
	}
	return string(buf)
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (g *Ghostparser) getUrlAndTagAndName(rel *Release, sc *goquery.Selection) {
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
					rel.addTag(sd.Text())
					i := g.getBoardId(attr)
					if i != -1 {
						rel.checkCat(i)
					} else {
						rel.Name = ""
						rel.Checksum = ""
					}
				}
			}
		}
	})
}

func (g *Ghostparser) getImageUrl(url string) (url2 string) {

	resp, err := g.Gc.Get(url)
	if err != nil {
		log.Error(err.Error())
		return url2
	}

	respbody, err := html.Parse(resp.Body)
	doc := goquery.NewDocumentFromNode(respbody)

	doc.Find(".resizeImage").Each(func(a int, sa *goquery.Selection) {
		if a == 0 {
			if attr, exist := sa.Attr("src"); exist {
				url2 = attr
			}
		}
	})

	return url2
}

//download image from town
func (g *Ghostparser) downloadImage(url string, name string) {
	imgurl := g.getImageUrl(url)
	if imgurl == "" {
		return
	}
	exist, err := exists("templates/images/" + name + ".jpg")
	if err != nil {
		log.Error(err.Error())
	}

	if !exist {
		resp, err := g.Gc.Get(imgurl)
		if err != nil {
			log.Error("image download failed, name: %v, url: %v", name, imgurl)
			log.Error(err.Error())
			return
		}
		if strings.Contains(imgurl, "jpg") || strings.Contains(imgurl, "jpeg") {
			img, err := jpeg.Decode(resp.Body)
			if err != nil {
				log.Error(err.Error())
				return
			}
			m := resize.Resize(300, 0, img, resize.Lanczos2Lut)
			out, err := os.Create("templates/images/" + name + ".jpg")
			if err != nil {
				log.Info(err.Error())
				return
			}

			// write new image to file
			jpeg.Encode(out, m, nil)
			out.Close()
		} else {
			log.Info(imgurl)
			img, err := png.Decode(resp.Body)
			if err != nil {
				log.Error(err.Error())
				return
			}
			m := resize.Resize(300, 0, img, resize.Lanczos2Lut)
			out, err := os.Create("templates/images/" + name + ".png")
			if err != nil {
				log.Info(err.Error())
				return
			}

			// write new image to file
			jpeg.Encode(out, m, nil)
			out.Close()
		}
	}
	time.Sleep(200 * time.Millisecond)
	return
}

func (g *Ghostparser) clearUrl(url string) string {
	if strings.Contains(url, "sid") {
		astr := strings.Split(url, "&")
		return astr[0]
	}
	return url
}

func (g *Ghostparser) getBoardId(str string) int {
	regex, err := regexp.Compile("boardid=.+&")
	if err != nil {
		log.Info(err.Error())
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
		log.Info(err.Error())
		return -1
	}
	return i
}

func (g *Ghostparser) encodeName(name string) string {
	h := sha1.New()
	h.Write([]byte(name))
	return hex.EncodeToString(h.Sum(nil))
}

//parse the http resp from Townclient
func (g *Ghostparser) ParseReleases() error {
	log.Info("parsing %v", g.Url)

	resp, err := g.Gc.Get(g.Url)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	respbody, err := html.Parse(resp.Body)
	doc := goquery.NewDocumentFromNode(respbody)

	var rel Release
	doc.Find("table").Each(func(a int, sa *goquery.Selection) {
		if a == 10 { //get the right table
			sa.Find("tr").Each(func(b int, sb *goquery.Selection) {
				sb.Find("td").Each(func(c int, sc *goquery.Selection) {
					if c == 2 {
						rel = Release{}
						g.getUrlAndTagAndName(&rel, sc)

						if rel.Name != "" {
							rel.Time = time.Now().Unix()
							rel.Checksum = g.encodeName(rel.Url)
							rel.checkQual()
							if rel.Name != "" {
								g.downloadImage(rel.Url, rel.Checksum)
								g.addRelease(rel)
							}
						}
					}
				})
			})
		}
		if g.Count == 0 { //get page count
			if a == 51 {
				sa.Find("a").Each(func(d int, sd *goquery.Selection) {
					if d == 3 {
						g.Count, err = strconv.Atoi(sd.Text())
					}
				})
			}
		}
	})

	return nil
}

func (r *Release) checkQual() {
	if strings.Contains(r.Tag, "music") || strings.Contains(r.Tag, "games") {
		return
	}
	if strings.Contains(r.Name, "1080") {
		r.addTag("1080")
	} else if strings.Contains(r.Name, "720") {
		r.addTag("720")
	} else if strings.Contains(r.Name, "untouched") {
		r.addTag("untouched")
	} else if strings.Contains(r.Name, "3d") {
		r.addTag("3d")
	} else if !strings.Contains(r.Url, "720") && !strings.Contains(r.Url, "1080") && !strings.Contains(r.Url, "untouched") && !strings.Contains(r.Url, "3d") {
		r.addTag("sd")
	} else {
		r.addTag("sd")
	}
}

func (r *Release) checkCat(boardid int) {
	switch boardid {
	case 26:
		r.addTag("cinema")
	case 176:
		r.addTag("cinema")
		r.addTag("hd")
	case 28, 29, 59:
		r.addTag("cinema")
		r.addTag("sd")
	case 101:
		r.addTag("movies")
	case 124, 125, 127, 1, 4, 6, 7:
		r.addTag("movies")
		r.addTag("sd")
	case 157, 158, 159, 211, 212, 160, 119, 120, 121, 213, 214, 122, 150, 202, 203, 141:
		r.addTag("movies")
		r.addTag("hd")
	case 143, 144, 145, 146:
		r.addTag("movies")
		r.addTag("hd")
		r.addTag("eng")
	case 53:
		r.addTag("series")
	case 77, 65, 215:
		r.addTag("series")
		r.addTag("sd")
	case 216, 217:
		r.addTag("series")
		r.addTag("hd")
	case 175, 208:
		r.addTag("series")
		r.addTag("hd")
		r.addTag("eng")
	case 225:
		r.addTag("series")
		r.addTag("sd")
		r.addTag("eng")
	case 12, 21, 19:
		r.addTag("music")
	case 166:
		r.addTag("music")
		r.addTag("rock")
	case 167:
		r.addTag("music")
		r.addTag("pop")
	case 168:
		r.addTag("music")
		r.addTag("jazz")
		r.addTag("blues")
		r.addTag("souls")
		r.addTag("country")
		r.addTag("reggae")
	case 169:
		r.addTag("music")
		r.addTag("hip-hop")
	case 170:
		r.addTag("music")
		r.addTag("electronic")
	case 171:
		r.addTag("music")
		r.addTag("schlager")
		r.addTag("volksmusik")
	case 172:
		r.addTag("music")
		r.addTag("oldies")
	case 173:
		r.addTag("music")
		r.addTag("metal")
	case 174:
		r.addTag("music")
		r.addTag("soundtrack")
	case 48:
		r.addTag("music")
		r.addTag("hörbuch")
	case 147:
		r.addTag("music")
		r.addTag("classic")
	case 177:
		r.addTag("music")
		r.addTag("discography")
	case 51:
		r.addTag("games")
		r.addTag("pc")
	case 32:
		r.addTag("games")
		r.addTag("xbox360")
	case 34:
		r.addTag("games")
		r.addTag("ps")
	case 37:
		r.addTag("games")
		r.addTag("wii")
	case 74:
		r.addTag("xxx")
	case 85:
		r.addTag("xxx")
		r.addTag("hd")
	case 70, 71, 133:
		r.addTag("xxx")
		r.addTag("sd")
	default:
		r.Name = ""
		r.Checksum = ""

	}
}

func (r *Release) addTag(tag string) {
	if r.Tag == "" {
		r.Tag = tag
	} else {
		r.Tag += "," + tag
	}
}

//adds a release to the Release slice of Townparser
func (g *Ghostparser) addRelease(r Release) {
	if len(g.Rel)+1 > cap(g.Rel) {
		temp := make([]Release, len(g.Rel), len(g.Rel)+1)
		copy(temp, g.Rel)
		g.Rel = temp
	}
	g.Rel = g.Rel[0 : len(g.Rel)+1]
	g.Rel[len(g.Rel)-1] = r
}
