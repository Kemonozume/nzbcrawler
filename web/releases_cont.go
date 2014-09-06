package web

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/Kemonozume/nzbcrawler/config"
	"github.com/jinzhu/gorm"
	"github.com/zenazn/goji/web"
)

type ReleasesController struct {
	releases *Releases
	cache    *Cache
}

func NewReleasesController(db *gorm.DB, conf *config.Config, cache *Cache) (rc *ReleasesController) {
	rc = &ReleasesController{}
	rc.releases = NewReleases(db, conf)
	rc.cache = cache
	return rc
}

func (rc *ReleasesController) GetRelease(c web.C, w http.ResponseWriter, r *http.Request) {
	idstr := c.URLParams["id"]

	id, err := strconv.Atoi(idstr)
	if err != nil {
		HandleError(w, r, err, "id parsing failed", http.StatusBadRequest, fmt.Sprintf("id = %s", idstr))
		return
	}

	by, err := rc.releases.GetReleaseWithId(int64(id))
	if err != nil {
		HandleError(w, r, err, "failed to get release", http.StatusBadRequest, fmt.Sprintf("id = %s", idstr))
		return
	}

	w.Header().Add("content-type", "application/json")
	w.Write(by)
}

func (rc *ReleasesController) ThankRelease(c web.C, w http.ResponseWriter, r *http.Request) {
	idstr := c.URLParams["id"]

	id, err := strconv.Atoi(idstr)
	if err != nil {
		HandleError(w, r, err, "id parsing failed", http.StatusBadRequest, fmt.Sprintf("id = %s", idstr))
		return
	}

	by, err := rc.releases.ThankReleaseWithId(int64(id))
	if err != nil {
		HandleError(w, r, err, "couldn't thank the release")
		return
	}

	w.Header().Add("content-type", "application/json")
	w.Write(by)
}

func (rc *ReleasesController) GetReleaseLink(c web.C, w http.ResponseWriter, r *http.Request) {
	idstr := c.URLParams["id"]

	id, err := strconv.Atoi(idstr)
	if err != nil {
		HandleError(w, r, err, "id parsing failed", http.StatusBadRequest, fmt.Sprintf("id = %s", idstr))
		return
	}

	url, err := rc.releases.GetReleaseLinkWithId(int64(id))
	if err != nil {
		HandleError(w, r, err, "link not found", http.StatusBadRequest, fmt.Sprintf("id = %s", idstr))
		return
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (rc *ReleasesController) GetReleaseImage(c web.C, w http.ResponseWriter, r *http.Request) {

	idstr := c.URLParams["id"]

	id, err := strconv.Atoi(idstr)
	if err != nil {
		HandleError(w, r, err, "id parsing failed", http.StatusBadRequest, fmt.Sprintf("id = %s", idstr))
		return
	}

	rel, err := rc.releases.GetReleaseImageWithId(int64(id))
	if err != nil {
		HandleError(w, r, err, "release not found", http.StatusBadRequest, fmt.Sprintf("id = %s", idstr))
		return
	}

	w.Header().Add("Cache-Control", "max-age=1296000")
	if rel.Image == "" {
		w.Write(i404)
	} else {
		req := NewRequest(rel.Image)
		rc.cache.Requests <- req
		w.Write(<-req.Response)
		close(req.Response)
	}

}

func (rc *ReleasesController) GetReleaseNzb(c web.C, w http.ResponseWriter, r *http.Request) {
	idstr := c.URLParams["id"]

	id, err := strconv.Atoi(idstr)
	if err != nil {
		HandleError(w, r, err, "id parsing failed", http.StatusBadRequest, fmt.Sprintf("id = %s", idstr))
		return
	}

	url, err := rc.releases.GetReleaseNzbWithId(int64(id))
	if err != nil {
		HandleError(w, r, err, "link not found", http.StatusBadRequest, fmt.Sprintf("id = %s", idstr))
		return
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (rc *ReleasesController) GetReleases(c web.C, w http.ResponseWriter, r *http.Request) {
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

	tags := querymap["tags"]
	name := querymap["name"]
	if len(tags) > 0 {
		by, err := rc.releases.GetReleasesWithTags(offset, tags)
		if err != nil {
			HandleError(w, r, err)
			return
		}
		w.Header().Add("content-type", "application/json")
		w.Write(by)
	} else if len(name) > 0 {
		by, err := rc.releases.GetReleasesWithName(offset, name[0])
		if err != nil {
			HandleError(w, r, err)
			return
		}
		w.Header().Add("content-type", "application/json")
		w.Write(by)
	} else {
		by, err := rc.releases.GetReleases(offset)
		if err != nil {
			HandleError(w, r, err)
			return
		}
		w.Header().Add("content-type", "application/json")
		w.Write(by)
	}

}
