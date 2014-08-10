package web

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"runtime"

	"github.com/Kemonozume/nzbcrawler/config"
	"github.com/Kemonozume/nzbcrawler/data"
	"github.com/gorilla/sessions"

	"github.com/gorilla/context"
	"github.com/jinzhu/gorm"
	"github.com/lidashuang/goji_gzip"
	log "github.com/sirupsen/logrus"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/graceful"
	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"
)

type Server struct {
	DB     *gorm.DB
	Config *config.Config
	Store  *sessions.CookieStore
}

const (
	TAG          = "[web]"
	LIMIT        = 100
	HITCOUNT     = "SELECT SUM(hits) FROm releases"
	TAGCOUNT     = "SELECT COUNT(*) FROM tags"
	RELEASECOUNT = "SELECT COUNT(*) FROM releases"
)

var i404 []byte

func (s *Server) Close() {
	graceful.Shutdown()
}

func (s *Server) Init() {

	//cookie store
	s.Store = sessions.NewCookieStore([]byte(s.Config.Secret))
	s.Store.Options = &sessions.Options{
		Path:   "/",
		Domain: "",
		MaxAge: 86400 * 7,
	}

	i404, _ = ioutil.ReadFile("templates/static/assets/img/404.jpg")

	goji.Use(gzip.GzipHandler)
	goji.Use(s.ServerMiddleWare)
	goji.Use(s.AuthMiddleWare)

	goji.Get("/assets/404.jpg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "image/jpeg")
		w.Write(i404)
	})
	goji.Get("/assets/*", http.FileServer(http.Dir("templates/static")))

	goji.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/index.html")
	})
	goji.Get("/logs", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/logs.html")
	})
	goji.Get("/stats", GetStats)

	goji.Get("/db/event/:id/", GetRelease)
	goji.Get("/db/event/:id/link", GetReleaseLink)
	goji.Get("/db/event/:id/nzb", GetReleaseNzb)
	goji.Get("/db/event/:id/thank", ThankRelease)
	goji.Get("/db/events/", GetReleases)
	goji.Get("/db/tags/", GetTags)
	goji.Get("/db/logs/", GetLogs)

	goji.Get("/login", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/login.html")
	})
	goji.Get("/login/:key", func(c web.C, w http.ResponseWriter, r *http.Request) {
		store := c.Env["store"].(*sessions.CookieStore)
		conf := c.Env["config"].(*config.Config)
		key := c.URLParams["key"]

		if key == conf.Key {
			session, _ := store.Get(r, "top-kek")
			session.Values["logged_in"] = true
			session.Save(r, w)
			log.Infof("%s %s managed to log in", TAG, r.RemoteAddr)
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		} else {
			log.Infof("%s %s failed to log in with key %s", TAG, r.RemoteAddr, key)
			http.Error(w, "nope", http.StatusForbidden)
		}
	})

	goji.NotFound(NotFound)

	goji.Abandon(middleware.Logger)

	addr := fmt.Sprintf("%s:%s", s.Config.Host, s.Config.Port)
	log.Infof("%s listening on %s", TAG, addr)
	err := graceful.ListenAndServe(addr, goji.DefaultMux)
	if err != nil {
		panic(err)
	}
	graceful.Wait()
	log.Infof("%s closing", TAG)
}

func HandleError(w http.ResponseWriter, r *http.Request, err error, details ...interface{}) bool {
	if err != nil {
		if len(details) > 2 {
			log.Errorf("%s %s | details: %s | %s", TAG, err.Error(), details[2].(string), r.URL.String())
		} else {
			log.Errorf("%s %s | %s", TAG, err.Error(), r.URL.String())
		}

		message := "Umm... have you tried turning it off and on again?"
		code := http.StatusInternalServerError

		if len(details) > 0 {
			message = details[0].(string)
		}
		if len(details) > 1 {
			code = details[1].(int)
		}

		http.Error(w, message, code)
		panic(err)
		return true
	}
	return false
}

func HandleRecovery() {
	if r := recover(); r != nil {
		log.Infof("%s recovered %v", TAG, r)
	}
}

func LogTime(url string, start time.Time) {
	log.Infof("%s GET %s in %s", TAG, url, time.Since(start))
}

//adds auth handler
func (s *Server) AuthMiddleWare(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		needs_auth := false
		if len(r.URL.Path) < 6 {
			needs_auth = true
		} else {
			str := r.URL.Path[1:6]
			if str != "login" && str != "asset" {
				needs_auth = true
			}
		}

		if needs_auth {
			logged_in := false
			sess, err := c.Env["store"].(*sessions.CookieStore).Get(r, "top-kek")
			if err != nil {
				log.Errorf("%s %s", TAG, err.Error())
			} else if sess != nil && sess.Values["logged_in"] != nil {
				if sess.Values["logged_in"].(bool) {
					logged_in = true
				}
				c.Env["store"].(*sessions.CookieStore).Save(r, w, sess)
			}
			if !logged_in {
				http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
				return
			}
		}

		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

//adds database and maybe other things later to the middlewarestack
func (s *Server) ServerMiddleWare(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer LogTime(r.URL.String(), time.Now())
		defer context.Clear(r)
		defer HandleRecovery()

		c.Env["store"] = s.Store
		c.Env["db"] = s.DB
		c.Env["config"] = s.Config
		c.Env["releases"] = Releases{}
		c.Env["logs"] = Logs{}
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

//custom 404 handler
func NotFound(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Umm... have you tried turning it off and on again?", 404)
}

func GetRelease(c web.C, w http.ResponseWriter, r *http.Request) {
	releases := c.Env["releases"].(Releases)
	idstr := c.URLParams["id"]

	id, err := strconv.Atoi(idstr)
	HandleError(w, r, err, "id parsing failed", http.StatusBadRequest, fmt.Sprintf("id = %s", idstr))

	by, err := releases.GetReleaseWithId(c, int64(id))
	HandleError(w, r, err, "failed to get release", http.StatusBadRequest, fmt.Sprintf("id = %s", idstr))

	w.Header().Add("content-type", "application/json")
	w.Write(by)
}

func ThankRelease(c web.C, w http.ResponseWriter, r *http.Request) {
	releases := c.Env["releases"].(Releases)
	idstr := c.URLParams["id"]

	id, err := strconv.Atoi(idstr)
	HandleError(w, r, err, "id parsing failed", http.StatusBadRequest, fmt.Sprintf("id = %s", idstr))

	by, err := releases.ThankReleaseWithId(c, int64(id))
	HandleError(w, r, err, "couldn't thank the release")

	w.Header().Add("content-type", "application/json")
	w.Write(by)

}

func GetReleaseLink(c web.C, w http.ResponseWriter, r *http.Request) {
	releases := c.Env["releases"].(Releases)
	idstr := c.URLParams["id"]

	id, err := strconv.Atoi(idstr)
	HandleError(w, r, err, "id parsing failed", http.StatusBadRequest, fmt.Sprintf("id = %s", idstr))

	url, err := releases.GetReleaseLinkWithId(c, int64(id))
	HandleError(w, r, err, "link not found", http.StatusBadRequest, fmt.Sprintf("id = %s", idstr))

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func GetReleaseNzb(c web.C, w http.ResponseWriter, r *http.Request) {
	releases := c.Env["releases"].(Releases)
	idstr := c.URLParams["id"]

	id, err := strconv.Atoi(idstr)
	HandleError(w, r, err, "id parsing failed", http.StatusBadRequest, fmt.Sprintf("id = %s", idstr))

	url, err := releases.GetReleaseNzbWithId(c, int64(id))
	HandleError(w, r, err, "link not found", http.StatusBadRequest, fmt.Sprintf("id = %s", idstr))

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func GetReleases(c web.C, w http.ResponseWriter, r *http.Request) {
	releases := c.Env["releases"].(Releases)

	offset := 0
	querymap := r.URL.Query()
	offsetmay := querymap["offset"]
	if len(offsetmay) > 0 {
		tmp, err := strconv.Atoi(offsetmay[0])
		HandleError(w, r, err, "bad offset", http.StatusBadRequest, fmt.Sprintf("offset = %s", offsetmay[0]))
		offset = tmp
	}

	tags := querymap["tags"]
	name := querymap["name"]
	if len(tags) > 0 {
		by, err := releases.GetReleasesWithTags(c, offset, tags)
		HandleError(w, r, err)
		w.Header().Add("content-type", "application/json")
		w.Write(by)
	} else if len(name) > 0 {
		by, err := releases.GetReleasesWithName(c, offset, name[0])
		HandleError(w, r, err)
		w.Header().Add("content-type", "application/json")
		w.Write(by)
	} else {
		by, err := releases.GetReleases(c, offset)
		HandleError(w, r, err)
		w.Header().Add("content-type", "application/json")
		w.Write(by)
	}

}

func GetLogs(c web.C, w http.ResponseWriter, r *http.Request) {
	logs := c.Env["logs"].(Logs)

	offset := 0
	querymap := r.URL.Query()
	offsetmay := querymap["offset"]
	if len(offsetmay) > 0 {
		tmp, err := strconv.Atoi(offsetmay[0])
		HandleError(w, r, err, "bad offset", http.StatusBadRequest, fmt.Sprintf("offset = %s", offsetmay[0]))
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

	by, err := logs.getLogsWithOptions(c, offset, options)
	HandleError(w, r, err)

	w.Header().Add("content-type", "application/json")
	w.Write(by)
}

func GetTags(c web.C, w http.ResponseWriter, r *http.Request) {
	db := c.Env["db"].(*gorm.DB)

	tags := []data.Tag{}
	err := db.Order("weight desc").Find(&tags).Error
	HandleError(w, r, err, "database request failed")

	by, err := json.Marshal(tags)
	HandleError(w, r, err, "json marshal failed")

	w.Header().Add("content-type", "application/json")
	w.Write(by)
}

type statwrap struct {
	Rel          []data.Release
	Ta           []data.Tag
	ReleaseCount int
	TagCount     int
	HitCount     int
	MemoryAcq    int
	MemoryUsed   int
	GoRoutines   int
}

func GetStats(c web.C, w http.ResponseWriter, r *http.Request) {
	db := c.Env["db"].(*gorm.DB)

	m := &runtime.MemStats{}
	runtime.ReadMemStats(m)
	stats := statwrap{}

	acq := m.Sys / 1024 / 1024
	used := m.Alloc / 1024 / 1024

	stats.GoRoutines = runtime.NumGoroutine()
	stats.MemoryAcq = int(acq)
	stats.MemoryUsed = int(used)

	err := db.Order("weight desc").Limit(10).Find(&stats.Ta).Error
	HandleError(w, r, err, "database request failed")

	err = db.Order("hits desc").Limit(10).Find(&stats.Rel).Error
	HandleError(w, r, err, "database request failed")

	row := db.Raw(HITCOUNT).Row()
	row.Scan(&stats.HitCount)

	row = db.Raw(TAGCOUNT).Row()
	row.Scan(&stats.TagCount)

	row = db.Raw(RELEASECOUNT).Row()
	row.Scan(&stats.ReleaseCount)

	tmp, err := template.ParseFiles("templates/stats.html")
	tmp.Execute(w, stats)
}
