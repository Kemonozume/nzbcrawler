package town

import (
	"bytes"
	"errors"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/Kemonozume/nzbcrawler/util"
	"github.com/PuerkitoBio/goquery"
)

type TownPostParser struct {
	doc *goquery.Document
}

var ErrorPostIdNotFound error = errors.New("no postid found")
var ErrorSecurityTokenNotFound error = errors.New("no securitytoken found")

//creating TownPostParser with a byte slice
func NewTownPostParserWithBytes(site []byte) (tp *TownPostParser, err error) {
	tp = &TownPostParser{}
	site = util.BytesToUtf8(site)
	tp.doc, err = goquery.NewDocumentFromReader(bytes.NewReader(site))
	return
}

//creating a TownPostParser using the TownClient and a url
func NewTownPostParserWithClient(url string, client *TownClient) (tp *TownPostParser, err error) {
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

	tp = &TownPostParser{}
	tp.doc, err = goquery.NewDocumentFromReader(bytes.NewReader(bv))
	return
}

func (tp *TownPostParser) GetSecurityToken() (string, error) {
	sval := ""
	tp.doc.Find("input").EachWithBreak(func(a int, sa *goquery.Selection) bool {
		if name, exists := sa.Attr("name"); exists {
			if name == "securitytoken" {
				if val, exists := sa.Attr("value"); exists {
					sval = val
					return false
				}
			}
		}
		return true
	})
	if sval == "" {
		return sval, ErrorSecurityTokenNotFound
	}

	return sval, nil
}

func (tp *TownPostParser) GetPostId() (string, error) {
	postid := ""
	/*sel := tp.doc.Find("table").Eq(22)
	if id, exists := sel.Attr("id"); exists {
		reg := regexp.MustCompile(`\w+?(\d+)`)
		res := reg.FindStringSubmatch(id)
		if len(res) == 2 {
			postid = res[1]
		}
	}
	*/
	tp.doc.Find("a").EachWithBreak(func(index int, sel *goquery.Selection) bool {
		str := sel.Text()
		if href, exists := sel.Attr("href"); exists {
			if str == "permalink" {
				res := regexp.MustCompile(`.*post(\d+)`).FindStringSubmatch(href)
				if len(res) > 0 {
					postid = res[1]
					return false
				}
			}
		}
		return true
	})

	if postid == "" {
		return postid, ErrorPostIdNotFound
	}
	return postid, nil
}

func (tp *TownPostParser) GetNzbUrl() (url string) {
	tp.doc.Find("a").EachWithBreak(func(index int, sel *goquery.Selection) bool {
		if href, exists := sel.Attr("href"); exists {
			if strings.Contains(href, ".nzb") {
				url = href
				return false
			}
		}
		return true
	})

	return
}

func (tp *TownPostParser) GetPassword() (password string) {
	tp.doc.Find("div .smallfont").EachWithBreak(func(index int, sel *goquery.Selection) bool {
		text := sel.Text()
		if strings.Contains(text, "Code") {
			password = sel.Next().Text()
			return false
		}
		return true
	})
	return
}
