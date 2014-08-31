package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/zenazn/goji/web"
)

type LogsController struct {
	logs *Logs
}

func NewLogsController(db *gorm.DB) (lc *LogsController) {
	lc = &LogsController{}
	lc.logs = NewLogs(db)
	return
}

func (lc *LogsController) GetLogs(c web.C, w http.ResponseWriter, r *http.Request) {
	offset := 0
	querymap := r.URL.Query()
	offsetmay := querymap["offset"]
	if len(offsetmay) > 0 {
		tmp, err := strconv.Atoi(offsetmay[0])
		if err != nil {
			HandleError(w, r, err, "bad offset", http.StatusBadRequest, fmt.Sprintf("offset = %s", offsetmay[0]))
			return
		}
		offset = tmp
	}

	tags := querymap["tag"]
	levels := querymap["level"]

	if len(tags) > 0 {
		tags = strings.Split(tags[0], ",")
	}

	if len(levels) > 0 {
		levels = strings.Split(levels[0], ",")
	}

	options := make(map[string][]string)
	options["tags"] = tags
	options["levels"] = levels

	by, err := lc.logs.getLogsWithOptions(c, offset, options)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	w.Header().Add("content-type", "application/json")
	w.Write(by)
}
