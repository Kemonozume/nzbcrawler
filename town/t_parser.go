package town

import (
	"crypto/sha1"
	"encoding/hex"
	"github.com/dustin/goquery"
	log "github.com/dvirsky/go-pylog/logging"
	"github.com/nfnt/resize"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"os"
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

type Townparser struct {
	Url  string
	Tc   *Townclient
	Rel  []Release
	Site []byte
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

func (t *Townparser) ParsePageCount() (int, error) {
	resp, err := t.Tc.Get(t.Url)
	if err != nil {
		log.Error(err.Error())
		return 0, err
	}

	bv, _ := ioutil.ReadAll(resp.Body)
	node, err := goquery.ParseString(toUtf8(bv))
	if err != nil {
		return 0, err
	}

	var count int
	var err2 error
	count = 0
	node.Find("a").Each(func(index int, element *goquery.Node) {
		if count != 0 {
			return
		}
		nodes := goquery.Nodes{element}
		val := nodes.Attr("title")
		if val != "" {
			if strings.Contains(val, "Letzte Seite") {
				url := nodes.Attr("href")
				if url != "" {
					aStr := strings.Split(url, "=")
					count, err2 = strconv.Atoi(aStr[len(aStr)-1])
				}
			}
		}
	})
	if err2 != nil {
		return 0, err2
	}
	return count, nil
}

//download image from town
func (t *Townparser) downloadImage(url string, name string) error {
	if url == "0" {
		return nil
	}
	exist, err := exists("templates/images/" + name + ".jpg")
	if err != nil {
		log.Error(err.Error())
	}

	if !exist {
		resp, err := t.Tc.Get(url)
		if err != nil {
			log.Error("image download failed, name: %v, url: %v", name, url)
			log.Error(err.Error())
			return err
		}
		if strings.Contains(url, "jpg") || strings.Contains(url, "jpeg") {
			img, _ := jpeg.Decode(resp.Body)
			m := resize.Resize(300, 0, img, resize.Lanczos2Lut)
			out, err := os.Create("templates/images/" + name + ".jpg")
			if err != nil {
				log.Info(err.Error())
				return nil
			}

			// write new image to file
			jpeg.Encode(out, m, nil)
			out.Close()
		} else {
			log.Info(url)
			img, err := png.Decode(resp.Body)
			if err != nil {
				log.Error(err.Error())
			}
			m := resize.Resize(300, 0, img, resize.Lanczos2Lut)
			out, err := os.Create("templates/images/" + name + ".png")
			if err != nil {
				log.Info(err.Error())
				return nil
			}

			// write new image to file
			jpeg.Encode(out, m, nil)
			out.Close()
		}
	}
	time.Sleep(200 * time.Millisecond)
	return nil
}

func (t *Townparser) encodeName(name string) string {
	h := sha1.New()
	h.Write([]byte(name))
	return hex.EncodeToString(h.Sum(nil))
}

//parse the http resp from Townclient
func (t *Townparser) ParseReleases(flush bool) error {
	log.Info("parsing %v", t.Url)
	if flush {
		t.Site = nil
	}

	if t.Site == nil {
		resp, err := t.Tc.Get(t.Url)
		if err != nil {
			log.Error(err.Error())
			return err
		}

		bv, _ := ioutil.ReadAll(resp.Body)
		t.Site = bv
	}
	node, err := goquery.ParseString(toUtf8(t.Site))
	if err != nil {
		return err
	}

	node.Find("#threadslist tbody tr").Each(func(index int, element *goquery.Node) {
		nodes := goquery.Nodes{element}
		var rel Release
		nodes.Find("td").Each(func(index int, element *goquery.Node) {
			nodes := goquery.Nodes{element}
			class := nodes.Attr("class")
			if strings.Contains(class, "alt") {
				switch index {
				case 0:
					rel = Release{}
				case 1:
					rel.Image = element.Child[1].Attr[2].Val
				case 2:
					nodes.Find("a").Each(func(i int, se *goquery.Node) {
						nodes2 := goquery.Nodes{se}
						switch i {
						case 1:
							text := nodes2.Text()
							if strings.Contains(text, "/") {
								text = strings.Join(strings.Split(text, "/"), ",")
							}
							text = strings.ToLower(text)
							rel.addTag(text)
						case 2:
							rel.Name = nodes2.Text()
							rel.Url = nodes2.Attr("href")
							rel.Checksum = t.encodeName(rel.Url)
						}
					})
				case 6:
					rel.Time = time.Now().Unix()
					rel.fillRelease()
					if rel.Checksum != "" {
						t.downloadImage(rel.Image, rel.Checksum)
						t.addRelease(rel)
					}
				}
			}
		})

	})

	return nil
}

func (r *Release) fillRelease() {
	url := r.Url
	subforum := strings.Split(url, "/")[0]
	r.checkCat(subforum)
	r.Url = "http://www.town.ag/v2/" + r.Url
	r.checkQual()
}

func (r *Release) addTag(tag string) {
	if r.Tag == "" {
		r.Tag = tag
	} else {
		r.Tag += "," + tag
	}
}

func (r *Release) checkQual() {
	if strings.Contains(r.Tag, "music") || strings.Contains(r.Tag, "games") {
		return
	}
	if strings.Contains(r.Name, "1080") {
		r.addTag("1080")
		r.addTag("hd")
	} else if strings.Contains(r.Name, "720") {
		r.addTag("720")
		r.addTag("hd")
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

func (r *Release) checkCat(subforum string) {
	switch subforum {
	case "cine-xvid-mvcd-svcd", "cine-dvd", "cine-hd-blu-ray":
		r.addTag("cinema")
	case "international-cine-corner":
		r.addTag("cinema")
		r.addTag("eng")
	case "720p-movies", "1080p-movies", "untouched", "doku-sport-musik", "3d-formate", "xvid", "dvd", "doku-xvid", "doku-dvd-hd":
		r.addTag("movies")
	case "720p-corner", "1080p-corner", "dvd-movies", "xvid-movies":
		r.addTag("movies")
		r.addTag("eng")
	case "kiddy-zone", "kids-serie", "aktuelle-serien-hd-xvid", "komplette-staffeln":
		r.addTag("series")
	case "tv-series-complete", "tv-series":
		r.addTag("series")
		r.addTag("eng")
	case "anime-mangas":
		r.addTag("series")
		r.addTag("anime")
	case "xxx-0-day-englisch", "xxx-dvd-hd", "xxx-france-italia", "xxx-pix-siterips-clips", "xxx-englisch", "xxx-german", "xxx-other", "big-packs-over-5gb":
		r.addTag("xxx")
	case "xxx-asian":
		r.addTag("xxx")
		r.addTag("asian")
	case "spiele", "sonstiges":
		r.addTag("games")
		r.addTag("pc")
	case "xbox", "xbox360", "xbox360-sonstiges":
		r.addTag("games")
		r.addTag("xbox")
	case "playstation-1", "playstation-2", "playstation-3", "playstation-portable", "ps3-sonstiges":
		r.addTag("games")
		r.addTag("playstation")
		r.addTag("ps")
	case "nds-gba", "wii-gamecube", "wii-sonstiges":
		r.addTag("games")
		r.addTag("wii")
	case "alben", "discography", "hoerbuecher-allgemein", "klassiker-oldies", "charts", "musik-dvd-xvid", "musik-others":
		r.addTag("music")
	case "lossless":
		r.addTag("music")
		r.addTag("lossless")
	default:
		r.Checksum = ""

	}
}

//adds a release to the Release slice of Townparser
func (t *Townparser) addRelease(r Release) {
	if len(t.Rel)+1 > cap(t.Rel) {
		temp := make([]Release, len(t.Rel), len(t.Rel)+1)
		copy(temp, t.Rel)
		t.Rel = temp
	}
	t.Rel = t.Rel[0 : len(t.Rel)+1]
	t.Rel[len(t.Rel)-1] = r
}
