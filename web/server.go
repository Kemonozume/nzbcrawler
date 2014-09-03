package web

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/pprof"
	"strings"
	"text/template"
	"time"

	"runtime"
	"runtime/debug"

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
	Cache  *Cache
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
	CacheMB      int
	CacheCount   int
}

const (
	TAG          = "[web]"
	LIMIT        = 100
	HITCOUNT     = "SELECT SUM(hits) FROm releases"
	TAGCOUNT     = "SELECT COUNT(*) FROM tags"
	RELEASECOUNT = "SELECT COUNT(*) FROM releases"
)

var webdebug = flag.Bool("webdebug", false, "enables or disabls webdebug")
var i404 []byte
var stats *statwrap
var defaultime time.Time
var cl chan bool

func (s *Server) Close() {
	graceful.Shutdown()
	cl <- true
}

func (s *Server) watcher() {
	for cl != nil {
		select {
		case <-cl:
			cl = nil
			break
		case <-time.Tick(time.Second * 3):
			err := RefreshStats(s.Cache, s.DB)
			if err != nil {
				log.Error(err.Error())
			}
		}
	}
}

func (s *Server) Init() {
	cl = make(chan bool)

	i404, err := ioutil.ReadFile("templates/static/assets/img/404.jpg")
	if err != nil {
		panic(err)
	}

	//cookie store
	s.Store = sessions.NewCookieStore([]byte(s.Config.Secret))
	s.Store.Options = &sessions.Options{
		Path:   "/",
		Domain: "",
		MaxAge: 86400 * 7,
	}

	s.Cache = NewCache(s.Config.CacheSize*1024*1024, s.Config.CacheFree*1024*1024, true)

	stats = &statwrap{}

	go s.watcher()

	rc := NewReleasesController(s.DB, s.Config, s.Cache)
	lc := NewLogsController(s.DB)

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
	goji.Get("/stats", s.GetStats)
	goji.Get("/stats/free", s.Force)

	goji.Get("/db/release/:id/", rc.GetRelease)
	goji.Get("/db/release/:id/link", rc.GetReleaseLink)
	goji.Get("/db/release/:id/nzb", rc.GetReleaseNzb)
	goji.Get("/db/release/:id/image", rc.GetReleaseImage)
	goji.Get("/db/release/:id/thank", rc.ThankRelease)
	goji.Get("/db/releases/", rc.GetReleases)
	goji.Get("/db/tags/", s.GetTags)
	goji.Get("/db/logs/", lc.GetLogs)

	goji.Get("/login", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/login.html")
	})
	goji.Get("/login/:key", s.Login)

	if *webdebug {
		goji.Handle("/debug/pprof/", pprof.Index)
		goji.Handle("/debug/pprof/cmdline", pprof.Cmdline)
		goji.Handle("/debug/pprof/profile", pprof.Profile)
		goji.Handle("/debug/pprof/symbol", pprof.Symbol)
		goji.Handle("/debug/pprof/block", pprof.Handler("block").ServeHTTP)
		goji.Handle("/debug/pprof/heap", pprof.Handler("heap").ServeHTTP)
		goji.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP)
		goji.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
	}

	goji.NotFound(NotFound)

	goji.Abandon(middleware.Logger)

	addr := fmt.Sprintf("%s:%s", s.Config.Host, s.Config.Port)
	log.Infof("%s listening on %s", TAG, addr)

	se := &graceful.Server{
		Addr:           addr,
		Handler:        goji.DefaultMux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	err = se.ListenAndServe()
	if err != nil {
		panic(err)
	}

	graceful.Wait()
	log.Infof("%s closing", TAG)
}

func HandleError(w http.ResponseWriter, r *http.Request, err error, details ...interface{}) {
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
}

func HandleRecovery() {
	if r := recover(); r != nil {
		log.Infof("%s recovered %v", TAG, r)
	}
}

func LogTime(url string, start time.Time) {
	if strings.Contains(url, "/image") || strings.Contains(url, "/assets/") {
		return
	}
	log.Infof("%s GET %s in %s", TAG, url, time.Since(start))
}

func (s *Server) Login(c web.C, w http.ResponseWriter, r *http.Request) {
	key := c.URLParams["key"]
	if key == s.Config.Key {
		session, _ := s.Store.Get(r, "top-kek")
		session.Values["logged_in"] = true
		session.Values["ip"] = strings.Split(r.RemoteAddr, ":")[0]
		session.Save(r, w)
		log.Warningf("%s %s managed to log in", TAG, r.RemoteAddr)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	} else {
		log.Warningf("%s %s failed to log in with key %s", TAG, r.RemoteAddr, key)
		http.Error(w, "nope", http.StatusForbidden)
	}
}

//adds auth handler
func (s *Server) AuthMiddleWare(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		needs_auth := false
		if len(r.URL.Path) < 6 {
			needs_auth = true
		} else {
			str := r.URL.Path[1:6]
			if str != "login" && str != "asset" && str != "debug" {
				needs_auth = true
			}
		}

		if needs_auth {
			logged_in := false
			sess, err := s.Store.Get(r, "top-kek")
			if err != nil {
				log.Errorf("%s %s", TAG, err.Error())
			} else if sess != nil && sess.Values["logged_in"] != nil && sess.Values["ip"] != nil {
				if sess.Values["logged_in"].(bool) && strings.Split(r.RemoteAddr, ":")[0] == sess.Values["ip"].(string) {
					logged_in = true
				}
				s.Store.Save(r, w, sess)
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
		now := time.Now()
		defer context.Clear(r)
		defer HandleRecovery()
		h.ServeHTTP(w, r)
		LogTime(r.URL.String(), now)
	}
	return http.HandlerFunc(fn)
}

//custom 404 handler
func NotFound(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Umm... have you tried turning it off and on again?", 404)
}

func (s *Server) GetTags(c web.C, w http.ResponseWriter, r *http.Request) {
	tags := []data.Tag{}
	err := s.DB.Order("weight desc").Find(&tags).Error
	if err != nil {
		HandleError(w, r, err, "database request failed")
		return
	}

	by, err := json.Marshal(tags)
	if err != nil {
		HandleError(w, r, err, "json marshal failed")
		return
	}

	w.Header().Add("content-type", "application/json")
	w.Write(by)
}

func RefreshStats(cache *Cache, db *gorm.DB) (err error) {
	m := &runtime.MemStats{}
	runtime.ReadMemStats(m)

	acq := m.Sys / 1024 / 1024
	used := m.Alloc / 1024 / 1024

	m = nil

	stats.GoRoutines = runtime.NumGoroutine()
	stats.MemoryAcq = int(acq)
	stats.MemoryUsed = int(used)

	stats.CacheMB = cache.GetSizeInMb()
	stats.CacheCount = cache.GetSize()

	stats.Rel = nil
	stats.Ta = nil

	err = db.Order("weight desc").Limit(10).Find(&stats.Ta).Error
	if err != nil {
		return
	}

	err = db.Order("hits desc").Limit(10).Find(&stats.Rel).Error
	if err != nil {
		return
	}

	row := db.Raw(HITCOUNT).Row()
	row.Scan(&stats.HitCount)

	row = db.Raw(TAGCOUNT).Row()
	row.Scan(&stats.TagCount)

	row = db.Raw(RELEASECOUNT).Row()
	row.Scan(&stats.ReleaseCount)
	return
}

func (s *Server) GetStats(c web.C, w http.ResponseWriter, r *http.Request) {
	bla := *stats
	t, err := template.ParseFiles("templates/stats.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		err = t.Execute(w, bla)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (s *Server) Force(c web.C, w http.ResponseWriter, r *http.Request) {
	runtime.GC()
	debug.FreeOSMemory()
	http.Redirect(w, r, "/stats", 302)
}
