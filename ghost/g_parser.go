package ghost

import (
	"crypto/sha1"
	"encoding/hex"
	"image/jpeg"
	"image/png"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"./../data"

	"code.google.com/p/go.net/html"
	"github.com/PuerkitoBio/goquery"
	log "github.com/dvirsky/go-pylog/logging"
	"github.com/nfnt/resize"
)

type Ghostparser struct {
	Url   string
	Gc    *Ghostclient
	Rel   []data.Release
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

func (g *Ghostparser) getUrlAndTagAndName(rel *data.Release, sc *goquery.Selection) {
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
					rel.AddTag(sd.Text())
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

func (g *Ghostparser) getImageUrl(url string) (url2 string) {

	resp, err := g.Gc.Get(url)
	if err != nil {
		log.Error("%s %s", TAG, err.Error())
		return url2
	}
	defer resp.Body.Close()

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
	defer func() {
		if err := recover(); err != nil {
			log.Info("%s recovered from panic", TAG)
			return
		}
	}()
	imgurl := g.getImageUrl(url)
	if imgurl == "" {
		return
	}
	exist, err := exists("templates/static/images/" + name + ".jpg")
	if err != nil {
		log.Error("%s %s", TAG, err.Error())
	}

	if !exist {
		resp, err := g.Gc.Get(imgurl)
		if err != nil {
			log.Error("%s image download failed, name: %v, url: %v", TAG, name, imgurl)
			log.Error("%s %s", TAG, err.Error())
			return
		}
		defer resp.Body.Close()
		if strings.Contains(imgurl, "jpg") || strings.Contains(imgurl, "jpeg") {
			img, err := jpeg.Decode(resp.Body)
			if err != nil {
				log.Error("%s %s", TAG, err.Error())
				return
			}
			m := resize.Resize(300, 0, img, resize.Lanczos2Lut)
			out, err := os.Create("templates/static/images/" + name + ".jpg")
			if err != nil {
				log.Error("%s %s", TAG, err.Error())
				return
			}

			// write new image to file
			jpeg.Encode(out, m, nil)
			out.Close()
		} else if strings.Contains(imgurl, "png") {
			img, err := png.Decode(resp.Body)
			if err != nil {
				log.Error("%s %s", TAG, err.Error())
				return
			}
			m := resize.Resize(300, 0, img, resize.Lanczos2Lut)
			out, err := os.Create("templates/static/images/" + name + ".png")
			if err != nil {
				log.Error("%s %s", TAG, err.Error())
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
		log.Error("%s %s", TAG, err.Error())
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
		log.Error("%s %s", TAG, err.Error())
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
	log.Info("%s parsing %v", TAG, g.Url)

	resp, err := g.Gc.Get(g.Url)
	if err != nil {
		log.Error("%s %s", TAG, err.Error())
		return err
	}
	defer resp.Body.Close()

	respbody, err := html.Parse(resp.Body)
	doc := goquery.NewDocumentFromNode(respbody)

	var rel data.Release
	doc.Find("table").Each(func(a int, sa *goquery.Selection) {
		if a == 10 { //get the right table
			sa.Find("tr").Each(func(b int, sb *goquery.Selection) {
				sb.Find("td").Each(func(c int, sc *goquery.Selection) {
					if c == 2 {
						rel = data.Release{}
						g.getUrlAndTagAndName(&rel, sc)

						if rel.Name != "" {
							rel.Time = time.Now().Unix()
							rel.Checksum = g.encodeName(rel.Url)
							g.checkQual(&rel)
							if rel.Name != "" {
								rel.Hits = 0
								rel.Rating = 0
								g.downloadImage(rel.Url, rel.Checksum)
								g.Rel = append(g.Rel, rel)
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

func (g *Ghostparser) checkQual(r *data.Release) {
	if strings.Contains(r.Tag, "music") || strings.Contains(r.Tag, "games") {
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
	} else {
		r.AddTag("sd")
	}
}

func (g *Ghostparser) checkCat(r *data.Release, boardid int) {
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
		r.Name = ""
		r.Checksum = ""

	}
}
