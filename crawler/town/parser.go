package town

import (
	"bytes"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Kemonozume/nzbcrawler/crawler"
	"github.com/Kemonozume/nzbcrawler/data"
	"github.com/Kemonozume/nzbcrawler/util"
	"github.com/PuerkitoBio/goquery"
)

type TownParser struct {
	doc *goquery.Document
	Rel []data.Release
}

//creating TownParser with a byte slice
func NewTownParserWithBytes(site []byte) (tp *TownParser, err error) {
	tp = &TownParser{}
	site = util.BytesToUtf8(site)
	tp.doc, err = goquery.NewDocumentFromReader(bytes.NewReader(site))
	return
}

//creating a TownPostParser using the TownClient and a url
func NewTownParserWithClient(url string, client *TownClient) (tp *TownParser, err error) {
	if !client.IsLoggedIn() {
		err = client.Login()
		if err != nil {
			return
		}
	}

	resp, err := client.Get(url)
	if err != nil {
		return
	}

	defer resp.Body.Close()

	bv, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	bv = util.BytesToUtf8(bv)

	tp = &TownParser{}
	tp.doc, err = goquery.NewDocumentFromReader(bytes.NewReader(bv))
	return
}

func (tp *TownParser) ParseUrlWithClient(url string, client *crawler.Client) (err error) {
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

	bv, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	bv = util.BytesToUtf8(bv)

	tp.doc, err = goquery.NewDocumentFromReader(bytes.NewReader(bv))
	return
}

//appends a new Release to the Release slice
func (tp *TownParser) append(rel data.Release) {
	tp.Rel = append(tp.Rel, rel)
}

func (tp *TownParser) GetMaxPage() (pagec int) {
	pagec = -1
	tp.doc.Find("a").EachWithBreak(func(index int, sel *goquery.Selection) bool {
		if title, exists := sel.Attr("title"); exists {
			if strings.Contains(title, "Letzte Seite") {
				if href, exists2 := sel.Attr("href"); exists2 {
					res := regexp.MustCompile(`.*page=(\d+)`).FindStringSubmatch(href)
					if len(res) > 0 {
						if l, err := strconv.Atoi(res[1]); err == nil {
							pagec = l
							return false
						}
					}
				}
			}
		}
		return true
	})
	return
}

//iterates over all releases in the document provided
func (tp *TownParser) ParseReleases() []data.Release {
	sel := tp.doc.Find("#threadslist tbody tr")
	//log.Info("%s found %d topics", TAG, sel.Size())
	sel.Each(tp.parseRelease)
	return tp.Rel
}

//used to parse each release
func (tp *TownParser) parseRelease(index int, element *goquery.Selection) {
	var rel data.Release
	element.Find("td").Each(func(index2 int, element2 *goquery.Selection) {
		if class, exists := element2.Attr("class"); exists {
			if strings.Contains(class, "alt") {
				switch index2 {
				case 0:
					rel = data.Release{}
				case 1:
					if imgsrc, exists := element2.Children().First().Attr("src"); exists {
						rel.Image = imgsrc
					}
				case 2:
					element2.Find("a").Each(func(i int, element3 *goquery.Selection) {
						switch i {
						case 1:
							text := element3.Text()
							if text != "" {
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
							}
						case 2:
							rel.Name = element3.Text()
							if href, exists := element3.Attr("href"); exists {
								rel.Url = href
							}
							rel.EncodeName()
						}
					})
				case 6:
					rel.Time = time.Now().Unix()
					tp.fillRelease(&rel)
					if rel.Checksum != "" {
						rel.Hits = 0
						tp.append(rel)
					}
				}
			}
		}
	})
}

//filling missing Informations like Quality and Category
func (tp *TownParser) fillRelease(r *data.Release) {
	url := r.Url
	subforum := strings.Split(url, "/")[0]
	tp.checkCat(r, subforum)
	r.Url = "http://www.town.ag/v2/" + r.Url
	tp.checkQual(r)
}

//adds tags based on the Name/Url
func (tp *TownParser) checkQual(r *data.Release) {
	if r.HasTag("music") || r.HasTag("games") {
		return
	}
	if strings.Contains(r.Name, "1080") {
		r.AddTag("1080")
		r.AddTag("hd")
	} else if strings.Contains(r.Name, "720") {
		r.AddTag("720")
		r.AddTag("hd")
	} else if strings.Contains(r.Name, "untouched") {
		r.AddTag("untouched")
	} else if strings.Contains(r.Name, "3d") {
		r.AddTag("3d")
	} else if !strings.Contains(r.Url, "720") && !strings.Contains(r.Url, "1080") && !strings.Contains(r.Url, "untouched") && !strings.Contains(r.Url, "3d") {
		r.AddTag("sd")
	}
}

//adds Tags based on the subforum category
func (tp *TownParser) checkCat(r *data.Release, subforum string) {
	switch subforum {
	case "cine-xvid-mvcd-svcd", "cine-dvd", "cine-hd-blu-ray":
		r.AddTag("cinema")
	case "international-cine-corner":
		r.AddTag("cinema")
		r.AddTag("eng")
	case "720p-movies", "1080p-movies", "untouched", "doku-sport-musik", "3d-formate", "xvid", "dvd", "doku-xvid", "doku-dvd-hd":
		r.AddTag("movies")
	case "720p-corner", "1080p-corner", "dvd-movies", "xvid-movies":
		r.AddTag("movies")
		r.AddTag("eng")
	case "kiddy-zone", "kids-serie", "aktuelle-serien-hd-xvid", "komplette-staffeln":
		r.AddTag("series")
	case "tv-series-complete", "tv-series":
		r.AddTag("series")
		r.AddTag("eng")
	case "anime-mangas":
		r.AddTag("series")
		r.AddTag("anime")
	case "xxx-0-day-englisch", "xxx-dvd-hd", "xxx-france-italia", "xxx-pix-siterips-clips", "xxx-englisch", "xxx-german", "xxx-other", "big-packs-over-5gb":
		r.AddTag("xxx")
	case "xxx-asian":
		r.AddTag("xxx")
		r.AddTag("asian")
	case "spiele", "sonstiges":
		r.AddTag("games")
		r.AddTag("pc")
	case "xbox", "xbox360", "xbox360-sonstiges":
		r.AddTag("games")
		r.AddTag("xbox")
	case "playstation-1", "playstation-2", "playstation-3", "playstation-portable", "ps3-sonstiges":
		r.AddTag("games")
		r.AddTag("playstation")
		r.AddTag("ps")
	case "nds-gba", "wii-gamecube", "wii-sonstiges":
		r.AddTag("games")
		r.AddTag("wii")
	case "alben", "discography", "hoerbuecher-allgemein", "klassiker-oldies", "charts", "musik-dvd-xvid", "musik-others":
		r.AddTag("music")
	case "lossless":
		r.AddTag("music")
		r.AddTag("lossless")
	default:
		r.AddTag("no_cat")

	}
}
