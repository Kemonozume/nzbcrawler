package town

import (
	"crypto/sha1"
	"encoding/hex"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
	"./../data"

	"github.com/dustin/goquery"
	log "github.com/dvirsky/go-pylog/logging"
	"github.com/nfnt/resize"
)

type Townparser struct {
	Url  string
	Tc   *Townclient
	Rel  []data.Release
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
		log.Error("%s %s", TAG, err.Error())
		return 0, err
	}
	defer resp.Body.Close()

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
	defer func() {
		if err := recover(); err != nil {
			log.Info("%s recovered from panic", TAG)
			return
		}
	}()
	if url == "0" {
		return nil
	}
	exist, err := exists("templates/static/images/" + name + ".jpg")
	if err != nil {
		log.Error("%s %s", TAG, err.Error())
	}

	if !exist {
		resp, err := t.Tc.Get(url)
		if err != nil {
			log.Error("%s image download failed, name: %v, url: %v", TAG, name, url)
			log.Error("%s %s", TAG, err.Error())
			return err
		}
		defer resp.Body.Close()
		if strings.Contains(url, "jpg") || strings.Contains(url, "jpeg") {
			img, _ := jpeg.Decode(resp.Body)
			m := resize.Resize(300, 0, img, resize.Lanczos2Lut)
			out, err := os.Create("templates/static/images/" + name + ".jpg")
			if err != nil {
				log.Error("%s %s", TAG, err.Error())
				return nil
			}

			// write new image to file
			jpeg.Encode(out, m, nil)
			out.Close()
		} else if strings.Contains(url, "png") {
			img, err := png.Decode(resp.Body)
			if err != nil {
				log.Error("%s %s", TAG, err.Error())
			}
			m := resize.Resize(300, 0, img, resize.Lanczos2Lut)
			out, err := os.Create("templates/static/images/" + name + ".png")
			if err != nil {
				log.Error("%s %s", TAG, err.Error())
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
	log.Info("%s parsing %v", TAG, t.Url)
	if flush {
		t.Site = nil
	}

	if t.Site == nil {
		resp, err := t.Tc.Get(t.Url)
		if err != nil {
			log.Error("%s %s", TAG, err.Error())
			return err
		}
		defer resp.Body.Close()

		bv, _ := ioutil.ReadAll(resp.Body)
		t.Site = bv
	}
	node, err := goquery.ParseString(toUtf8(t.Site))
	if err != nil {
		return err
	}

	node.Find("#threadslist tbody tr").Each(func(index int, element *goquery.Node) {
		nodes := goquery.Nodes{element}
		var rel data.Release
		nodes.Find("td").Each(func(index int, element *goquery.Node) {
			nodes := goquery.Nodes{element}
			class := nodes.Attr("class")
			if strings.Contains(class, "alt") {
				switch index {
				case 0:
					rel = data.Release{}
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
							rel.AddTag(text, &rel)
						case 2:
							rel.Name = nodes2.Text()
							rel.Url = nodes2.Attr("href")
							rel.Checksum = t.encodeName(rel.Url)
						}
					})
				case 6:
					rel.Time = time.Now().Unix()
					t.fillRelease(&rel)
					if rel.Checksum != "" {
						rel.Hits = 0
						rel.Rating = 0
						t.downloadImage(rel.Image, rel.Checksum)
						t.Rel = append(t.Rel, rel)
					}
				}
			}
		})

	})

	return nil
}

func (t *Townparser) fillRelease(r *data.Release) {
	url := r.Url
	subforum := strings.Split(url, "/")[0]
	t.checkCat(r, subforum)
	r.Url = "http://www.town.ag/v2/" + r.Url
	t.checkQual(r)
}

func (t *Townparser) checkQual(r *data.Release) {
	if strings.Contains(r.Tag, "music") || strings.Contains(r.Tag, "games") {
		return
	}
	if strings.Contains(r.Name, "1080") {
		r.AddTag("1080", r)
		r.AddTag("hd", r)
	} else if strings.Contains(r.Name, "720") {
		r.AddTag("720", r)
		r.AddTag("hd", r)
	} else if strings.Contains(r.Name, "untouched") {
		r.AddTag("untouched", r)
	} else if strings.Contains(r.Name, "3d") {
		r.AddTag("3d", r)
	} else if !strings.Contains(r.Url, "720") && !strings.Contains(r.Url, "1080") && !strings.Contains(r.Url, "untouched") && !strings.Contains(r.Url, "3d") {
		r.AddTag("sd", r)
	} else {
		r.AddTag("sd", r)
	}
}

func (t *Townparser) checkCat(r *data.Release, subforum string) {
	switch subforum {
	case "cine-xvid-mvcd-svcd", "cine-dvd", "cine-hd-blu-ray":
		r.AddTag("cinema", r)
	case "international-cine-corner":
		r.AddTag("cinema", r)
		r.AddTag("eng", r)
	case "720p-movies", "1080p-movies", "untouched", "doku-sport-musik", "3d-formate", "xvid", "dvd", "doku-xvid", "doku-dvd-hd":
		r.AddTag("movies", r)
	case "720p-corner", "1080p-corner", "dvd-movies", "xvid-movies":
		r.AddTag("movies", r)
		r.AddTag("eng", r)
	case "kiddy-zone", "kids-serie", "aktuelle-serien-hd-xvid", "komplette-staffeln":
		r.AddTag("series", r)
	case "tv-series-complete", "tv-series":
		r.AddTag("series", r)
		r.AddTag("eng", r)
	case "anime-mangas":
		r.AddTag("series", r)
		r.AddTag("anime", r)
	case "xxx-0-day-englisch", "xxx-dvd-hd", "xxx-france-italia", "xxx-pix-siterips-clips", "xxx-englisch", "xxx-german", "xxx-other", "big-packs-over-5gb":
		r.AddTag("xxx", r)
	case "xxx-asian":
		r.AddTag("xxx", r)
		r.AddTag("asian", r)
	case "spiele", "sonstiges":
		r.AddTag("games", r)
		r.AddTag("pc", r)
	case "xbox", "xbox360", "xbox360-sonstiges":
		r.AddTag("games", r)
		r.AddTag("xbox", r)
	case "playstation-1", "playstation-2", "playstation-3", "playstation-portable", "ps3-sonstiges":
		r.AddTag("games", r)
		r.AddTag("playstation", r)
		r.AddTag("ps", r)
	case "nds-gba", "wii-gamecube", "wii-sonstiges":
		r.AddTag("games", r)
		r.AddTag("wii", r)
	case "alben", "discography", "hoerbuecher-allgemein", "klassiker-oldies", "charts", "musik-dvd-xvid", "musik-others":
		r.AddTag("music", r)
	case "lossless":
		r.AddTag("music", r)
		r.AddTag("lossless", r)
	default:
		r.Checksum = ""

	}
}
