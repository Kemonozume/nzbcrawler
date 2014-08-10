package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Kemonozume/nzbcrawler/config"
	"github.com/Kemonozume/nzbcrawler/crawler/town"
	"github.com/Kemonozume/nzbcrawler/data"
	"github.com/jinzhu/gorm"
	"github.com/zenazn/goji/web"
)

type Releases struct {
}

func (r Releases) GetReleaseWithId(c web.C, id int64) (by []byte, err error) {
	db := c.Env["db"].(*gorm.DB)

	rel := data.Release{Id: id}
	err = db.Model(&rel).First(&rel).Related(&rel.Tags, "Tags").Error
	if err != nil {
		return
	}

	by, err = json.Marshal(rel)
	return
}

func (r Releases) ThankReleaseWithId(c web.C, id int64) ([]byte, error) {
	conf := c.Env["config"].(*config.Config)
	db := c.Env["db"].(*gorm.DB)

	var rel data.Release

	err := db.Where("id = ?", id).First(&rel).Error
	if err != nil {
		return nil, err
	}

	if rel.Nzb == "" {
		tc := town.NewClient()
		tc.User = conf.TownUser
		tc.Password = conf.TownPassword

		err = tc.Login()
		if err != nil {
			return nil, err
		}

		tp, err := town.NewTownPostParserWithClient(rel.Url, tc)
		if err != nil {
			return nil, err
		}

		postid, err := tp.GetPostId()
		if err != nil {
			return nil, err
		}

		token, err := tp.GetSecurityToken()
		if err != nil {
			return nil, err
		}

		err = tc.ThankPost(postid, token)
		if err != nil {
			return nil, err
		}

		tp, err = town.NewTownPostParserWithClient(rel.Url, tc)
		if err != nil {
			return nil, err
		}

		url := tp.GetNzbUrl()
		passwd := tp.GetPassword()

		db.Model(&rel).UpdateColumn("password", passwd).UpdateColumn("nzb", url)

		retval := map[string]interface{}{
			"id":       id,
			"nzb":      url,
			"passwort": passwd,
		}

		by, err := json.Marshal(retval)
		return by, err
	} else {
		retval := map[string]interface{}{
			"id":       id,
			"nzb":      rel.Nzb,
			"passwort": rel.Password,
		}

		by, err := json.Marshal(retval)
		return by, err
	}

}

func (r Releases) GetReleaseLinkWithId(c web.C, id int64) (url string, err error) {
	db := c.Env["db"].(*gorm.DB)

	rel := data.Release{Id: id}
	err = db.Model(&rel).First(&rel).UpdateColumn("hits", rel.Hits+1).Error
	if err != nil {
		return
	}

	return rel.Url, nil
}

func (r Releases) GetReleaseNzbWithId(c web.C, id int64) (url string, err error) {
	db := c.Env["db"].(*gorm.DB)

	rel := data.Release{Id: id}
	err = db.Model(&rel).First(&rel).UpdateColumn("hits", rel.Hits+1).Error
	if err != nil {
		return
	}

	return rel.Nzb, nil
}

func (r Releases) GetReleases(c web.C, offset int) (by []byte, err error) {
	db := c.Env["db"].(*gorm.DB)

	var release []data.Release
	err = db.Order("time desc").Limit(LIMIT).Offset(offset).Find(&release).Error
	if err != nil {
		return
	}

	for i, rel := range release {
		db.Model(&rel).Related(&rel.Tags, "Tags")
		release[i] = rel
	}

	by, err = json.Marshal(release)

	return
}

func (r Releases) q(str string) string {
	re := strings.NewReplacer("'", "''")
	return "'" + re.Replace(str) + "'"
}

func (r Releases) GetReleasesWithTags(c web.C, offset int, tags []string) (by []byte, err error) {
	db := c.Env["db"].(*gorm.DB)

	tags = strings.Split(tags[0], ",")

	var release []data.Release
	var args []interface{}
	var buffer bytes.Buffer

	buffer.WriteString("SELECT r.id FROM releases_tags rt, releases r, tags t WHERE rt.tag_id = t.id")
	buffer.WriteString(" AND (t.value in (")
	for i, tag := range tags {
		buffer.WriteString("?")
		args = append(args, tag)
		if i+1 != len(tags) {
			buffer.WriteString(", ")
		}
	}
	buffer.WriteString(")) AND r.id = rt.release_id GROUP BY r.id HAVING COUNT( r.id )=")
	buffer.WriteString(strconv.Itoa(len(tags)))
	buffer.WriteString(" ORDER BY r.time DESC LIMIT ")
	buffer.WriteString(strconv.Itoa(LIMIT))
	buffer.WriteString(" OFFSET ")
	buffer.WriteString(strconv.Itoa(offset))

	query := buffer.String()

	rows, err := db.Raw(query, args...).Rows()
	if err != nil {
		return
	}
	for rows.Next() {
		var rel data.Release
		rows.Scan(&rel.Id)
		release = append(release, rel)
	}
	rows.Close()
	for i, rel := range release {
		db.Model(&rel).First(&rel).Related(&rel.Tags, "Tags")
		release[i] = rel
	}

	by, err = json.Marshal(release)
	return
}

func (r Releases) GetReleasesWithName(c web.C, offset int, name string) (by []byte, err error) {
	db := c.Env["db"].(*gorm.DB)

	var release []data.Release
	err = db.Where("name LIKE ?", fmt.Sprintf("%%%s%%s", name)).Order("time desc").Limit(LIMIT).Offset(offset).Find(&release).Error
	if err != nil {
		return
	}

	for i, rel := range release {
		db.Model(&rel).Related(&rel.Tags, "Tags")
		release[i] = rel
	}
	by, err = json.Marshal(release)
	return
}
