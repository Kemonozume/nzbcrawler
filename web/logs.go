package web

import (
	"bytes"
	"encoding/json"
	"strconv"

	"github.com/Kemonozume/nzbcrawler/data"
	"github.com/jinzhu/gorm"
	"github.com/zenazn/goji/web"
)

type Logs struct {
}

func (l Logs) GetLogs(c web.C, offset int) (by []byte, err error) {
	db := c.Env["db"].(*gorm.DB)

	logs := []data.Log{}
	err = db.Order("id desc").Limit(LIMIT).Offset(offset).Find(&logs).Error
	if err != nil {
		return
	}

	by, err = json.Marshal(logs)
	return
}

func (l Logs) getLogsWithOptions(c web.C, offset int, options map[string][]string) (by []byte, err error) {
	db := c.Env["db"].(*gorm.DB)

	var levels []string
	var tags []string
	var logs []data.Log
	var buffer bytes.Buffer
	var args []interface{}

	levels = options["levels"]
	tags = options["tags"]

	buffer.WriteString("SELECT id, level, time, tag, message FROM logs ")
	for i, tag := range tags {
		if i == 0 {
			buffer.WriteString("WHERE (")
		}
		buffer.WriteString("tag = ? ")
		if i+1 != len(tags) && len(tags) > 1 {
			buffer.WriteString("OR ")
		} else {
			buffer.WriteString(") ")
		}
		args = append(args, tag)
	}

	for i, level := range levels {
		if i == 0 {
			if len(tags) != 0 {
				buffer.WriteString("AND ")
			} else {
				buffer.WriteString("WHERE ")
			}
			buffer.WriteString("(")
		}
		buffer.WriteString("level = ? ")
		if i+1 != len(levels) {
			buffer.WriteString("OR ")
		} else {
			buffer.WriteString(")")
		}
		args = append(args, level)
	}
	buffer.WriteString("ORDER BY id DESC ")
	buffer.WriteString("LIMIT ")
	buffer.WriteString(strconv.Itoa(LIMIT))
	buffer.WriteString(" OFFSET ")
	buffer.WriteString(strconv.Itoa(offset))

	query := buffer.String()

	rows, err := db.Raw(query, args...).Rows()
	if err != nil {
		return
	}
	for rows.Next() {
		var rel data.Log
		rows.Scan(&rel.Id, &rel.Level, &rel.Time, &rel.Tag, &rel.Message)
		logs = append(logs, rel)
	}

	by, err = json.Marshal(logs)
	return
}
