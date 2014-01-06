package webserv

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"./../mydb"
	"./../town"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/sessions"
	"github.com/coopernurse/gorp"
	log "github.com/dvirsky/go-pylog/logging"
	"github.com/robfig/config"
)

type Server struct {
	Config  *config.Config
	Config2 *Conf
	RelDB   *gorp.DbMap
	LogDB   *gorp.DbMap
}

type Conf struct {
	Host          string
	Port          string
	TownName      string
	TownPassword  string
	GhostName     string
	GhostPassword string
	Key           string
	Secret        string
	Timeout       string
	Crawl         bool
}

var runner *Runner

func (s *Server) Init() {

	s.readConfig()

	store := sessions.NewCookieStore([]byte(s.Config2.Secret))
	store.Options(sessions.Options{
		Path:   "/",
		Domain: "",
		MaxAge: 86400 * 7,
	})

	//start Runner...
	if s.Config2.Crawl {
		runner = &Runner{s}
		go runner.Start()
	}

	m := martini.New()

	m.Map(s)

	m.Use(martini.Recovery())
	m.Use(martini.Static("templates/static"))
	m.Use(sessions.Sessions("top-kek", store))

	r := martini.NewRouter()
	r.Get("/", Auth, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/index.html")
	})
	r.Get("/db/events/:offset/:tags/:name", Auth, GetReleaseWithTagAndName)
	r.Get("/db/event/:checksum/link", Auth, LinkFollow)
	r.Get("/db/event/:checksum/score/:score", Auth, LinkFollow)
	r.Get("/log", Auth, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/log.html")
	})
	r.Get("/log/:offset/", Auth, GetLogs)
	r.Get("/log/:offset/:level", Auth, GetLogsWithLevel)
	r.Post("/log/clearlogs", Auth, func(server *Server) string {
		server.LogDB.Exec("drop table log")
		server.LogDB.AddTableWithName(mydb.Log{}, "log").SetKeys(true, "Uid")
		server.LogDB.CreateTablesIfNotExists()
		server.LogDB.Exec("vacuum")
		return ""
	})

	r.Get("/key/:key", func(res http.ResponseWriter, req *http.Request, server *Server, session sessions.Session, parms martini.Params) {
		key := parms["key"]
		if key == server.Config2.Key {
			log.Info("Login success from %v", req.RemoteAddr)
			session.Set("login", true)
			http.Redirect(res, req, "/", http.StatusFound)
		} else {
			log.Info("Login fail from %v", req.RemoteAddr)
			http.Error(res, "forbidden", http.StatusForbidden)
		}
	})

	m.Action(r.Handle)

	log.Info("listening on %v:%v", s.Config2.Host, s.Config2.Port)
	http.ListenAndServe(s.Config2.Host+":"+s.Config2.Port, m)
}

func Auth(res http.ResponseWriter, req *http.Request, session sessions.Session) {
	fmt.Printf("%v\n", session)
	if session.Get("login") != true {
		http.Error(res, "forbidden", http.StatusForbidden)
	}
}

func GetReleaseWithTagAndName(server *Server, parms martini.Params) string {
	offset := parms["offset"]
	tags := parms["tags"]
	name := parms["name"]
	command := ""
	if tags == "none" && name != "none" {
		command = "select * from release where name LIKE '%" + name + "%' ORDER BY time DESC LIMIT 200 OFFSET " + offset
	} else if name == "none" && tags != "none" {
		cb := &CMDBuilder{}
		command = cb.Tokenize(tags)
		command += " ORDER BY time DESC LIMIT 200 OFFSET " + offset
	} else if name != "none" && tags != "none" {
		cb := &CMDBuilder{}
		command = cb.Tokenize(tags)
		command += "AND name LIKE '%" + name + "%' ORDER BY time DESC LIMIT 200 OFFSET " + offset
	}

	var b []town.Release

	if command != "" {
		_, err := server.RelDB.Select(&b, command)
		if err != nil {
			log.Error(err.Error())
		}
	} else {
		_, err := server.RelDB.Select(&b, "select * from release ORDER BY time DESC LIMIT 200 OFFSET ?", offset)
		if err != nil {
			log.Error(err.Error())
		}
	}
	by, err := json.Marshal(b)
	if err != nil {
		log.Error(err.Error())
	}

	return string(by)
}

func LinkFollow(w http.ResponseWriter, r *http.Request, server *Server, parms martini.Params) {
	checksum := parms["checksum"]

	hits, err := server.RelDB.SelectInt("select hits from release where checksum=?", checksum)
	if err != nil {
		log.Error(err.Error())
	}

	oldhits := hits
	hits += 1
	log.Info("increasing hits for %s from %v to %v", checksum, oldhits, hits)
	_, err = server.RelDB.Exec("update release set hits=? where checksum=?", hits, checksum)
	if err != nil {
		log.Error(err.Error())
	}

	url, err := server.RelDB.SelectStr("select url from release where checksum=?", checksum)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func SetRating(server *Server, parms martini.Params) string {
	checksum := parms["checksum"]
	score, _ := strconv.Atoi(parms["score"])

	rating, err := server.RelDB.SelectInt("select rating from release where checksum=?", checksum)
	if err != nil {
		log.Error(err.Error())
	}
	oldrating := rating
	rating += int64(score)
	log.Info("changing score for %s from %v to %v", checksum, oldrating, rating)
	_, err = server.RelDB.Exec("update release set rating=? where checksum=?", rating, checksum)
	if err != nil {
		log.Error(err.Error())
	}

	return fmt.Sprintf("%v %v", checksum, score)
}

func GetLogs(server *Server, parms martini.Params) string {
	offset, _ := strconv.Atoi(parms["offset"])

	var all []mydb.Log
	_, err := server.LogDB.Select(&all, "select * from log ORDER BY id DESC LIMIT 50 OFFSET ?", offset)
	if err != nil {
		log.Error(err.Error())
	}
	by, err := json.Marshal(all)
	if err != nil {
		log.Error(err.Error())
	}

	return string(by)

}

func GetLogsWithLevel(server *Server, parms martini.Params) string {
	offset, _ := strconv.Atoi(parms["offset"])
	level := parms["level"]

	var all []mydb.Log
	_, err := server.LogDB.Select(&all, "select * from log WHERE Lvl = ? ORDER BY id DESC LIMIT 50 OFFSET ?", level, offset)
	if err != nil {
		log.Error(err.Error())
	}

	by, err := json.Marshal(all)
	if err != nil {
		log.Error(err.Error())
	}

	return string(by)
}

func (s *Server) readConfig() {
	s.Config2 = &Conf{}
	s.Config2.Port, _ = s.Config.String("default", "port")
	s.Config2.Host, _ = s.Config.String("default", "host")
	s.Config2.TownName, _ = s.Config.String("default", "town-name")
	s.Config2.TownPassword, _ = s.Config.String("default", "town-password")
	s.Config2.GhostName, _ = s.Config.String("default", "ghost-name")
	s.Config2.GhostPassword, _ = s.Config.String("default", "ghost-password")
	s.Config2.Key, _ = s.Config.String("default", "key")
	s.Config2.Secret, _ = s.Config.String("default", "secret")
	s.Config2.Crawl, _ = s.Config.Bool("default", "crawl")
	s.Config2.Timeout, _ = s.Config.String("default", "timeout")
}
