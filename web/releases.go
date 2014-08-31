package web

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/Kemonozume/nzbcrawler/config"
	"github.com/Kemonozume/nzbcrawler/crawler/town"
	"github.com/Kemonozume/nzbcrawler/data"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
)

type Releases struct {
	db   *gorm.DB
	conf *config.Config
}

func NewReleases(db *gorm.DB, conf *config.Config) (rel *Releases) {
	rel = &Releases{}
	rel.db = db
	rel.conf = conf
	return
}

func (r *Releases) GetReleaseWithId(id int64) (by []byte, err error) {
	rel := data.Release{Id: id}
	err = r.db.Model(&rel).First(&rel).Related(&rel.Tags, "Tags").Error
	if err != nil {
		return
	}

	by, err = json.Marshal(rel)
	return
}

func (r *Releases) GetReleaseImageWithId(id int64) (rel data.Release, err error) {
	rel = data.Release{Id: id}
	err = r.db.Model(&rel).First(&rel).Error
	return
}

func (r *Releases) ThankReleaseWithId(id int64) ([]byte, error) {
	var rel data.Release

	err := r.db.Where("id = ?", id).First(&rel).Error
	if err != nil {
		return nil, err
	}

	if rel.Nzb == "" {
		tc := town.NewClient()
		tc.User = r.conf.TownUser
		tc.Password = r.conf.TownPassword

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

		err = r.db.Model(&rel).First(&rel).UpdateColumns(data.Release{Nzb: url, Password: passwd}).Error
		if err != nil {
			return nil, err
		}

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

func (r *Releases) GetReleaseLinkWithId(id int64) (url string, err error) {
	rel := data.Release{Id: id}
	err = r.db.Model(&rel).First(&rel).UpdateColumn("hits", rel.Hits+1).Error
	if err != nil {
		return
	}

	return rel.Url, nil
}

func (r *Releases) GetReleaseNzbWithId(id int64) (url string, err error) {
	rel := data.Release{Id: id}
	err = r.db.Model(&rel).First(&rel).UpdateColumn("hits", rel.Hits+1).Error
	if err != nil {
		return
	}

	return rel.Nzb, nil
}

func (r *Releases) GetReleases(offset int) (by []byte, err error) {
	var release []data.Release
	err = r.db.Order("time desc").Limit(LIMIT).Offset(offset).Find(&release).Error
	if err != nil {
		return
	}

	for i, rel := range release {
		r.db.Model(&rel).Related(&rel.Tags, "Tags")
		release[i] = rel
	}

	by, err = json.Marshal(release)

	return
}

func (r *Releases) q(str string) string {
	re := strings.NewReplacer("'", "''")
	return "'" + re.Replace(str) + "'"
}

func (r *Releases) GetReleasesWithTags(offset int, tags []string) (by []byte, err error) {
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

	log.Infof("%s %s", "[sql]", query)

	rows, err := r.db.Raw(query, args...).Rows()
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
		r.db.Model(&rel).First(&rel).Related(&rel.Tags, "Tags")
		release[i] = rel
	}

	by, err = json.Marshal(release)
	return
}

func (r *Releases) GetReleasesWithName(offset int, name string) (by []byte, err error) {
	var buffer bytes.Buffer
	var release []data.Release

	buffer.WriteString("SELECT id FROM releases WHERE name LIKE '%")
	buffer.WriteString(name)
	buffer.WriteString("%' ORDER BY time DESC LIMIT 100 OFFSET ")
	buffer.WriteString(strconv.Itoa(offset))

	query := buffer.String()
	log.Infof("%s %s", "[sql]", query)

	rows, err := r.db.Raw(query).Rows()
	if err != nil {
		return
	}
	for rows.Next() {
		rel := data.Release{}
		rows.Scan(&rel.Id)
		release = append(release, rel)
	}
	rows.Close()

	for i, rel := range release {
		err = r.db.Where("id = ?", rel.Id).First(&rel).Error
		if err != nil {
			return
		}
		err = r.db.Model(&rel).Related(&rel.Tags, "Tags").Error
		if err != nil {
			return
		}
		release[i] = rel
	}
	by, err = json.Marshal(release)
	return
}
