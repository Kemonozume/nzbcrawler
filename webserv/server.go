package webserv

import (
	"./../mydb"
	"./../town"
	"encoding/json"
	"fmt"
	"github.com/coopernurse/gorp"
	log "github.com/dvirsky/go-pylog/logging"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/robfig/config"
	"net/http"
	"strconv"
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

var server *Server
var runner *Runner
var store *sessions.CookieStore

func (s *Server) Init() {

	s.readConfig()

	store = sessions.NewCookieStore([]byte(s.Config2.Secret))
	store.Options = &sessions.Options{
		Path:   "/",
		Domain: "",
		MaxAge: 86400 * 7,
	}

	//start Runner...
	if s.Config2.Crawl {
		runner = &Runner{s}
		go runner.Start()
	}

	server = s

	r := mux.NewRouter()

	//browse
	r.Handle("/", MyHandler{func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/index.html")
	}})
	r.Handle("/db/events/{offset:[0-9]+}/{tags}/{name}", MyHandler{GetReleaseWithTagAndName})
	r.Handle("/db/event/{checksum}/link", MyHandler{LinkFollow})
	r.Handle("/db/event/{checksum}/score/{score}", MyHandler{SetRating})

	//logs
	r.Handle("/log", MyHandler{func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/log.html")
	}})
	r.Handle("/log/{offset:[0-9]+}", MyHandler{GetLogs}).Methods("GET")
	r.Handle("/log/{offset:[0-9]+}/{level}", MyHandler{GetLogsWithLevel}).Methods("GET")
	r.Handle("/log/clearlogs", MyHandler{func(w http.ResponseWriter, r *http.Request) {
		server.LogDB.Exec("drop table log")
		server.LogDB.AddTableWithName(mydb.Log{}, "log").SetKeys(true, "Uid")
		server.LogDB.CreateTablesIfNotExists()
		server.LogDB.Exec("vacuum")
		fmt.Fprintf(w, "")
	}}).Methods("POST")

	//assets
	r.Handle("/public/{file:.+}", MyHandler{func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		file := vars["file"]
		http.ServeFile(w, r, "templates/assets/"+file)
	}})
	r.Handle("/images/{file:.+}", MyHandler{func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		file := vars["file"]
		w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, public, must-revalidate, proxy-revalidate", 432000))
		w.Header().Add("Content-Type", "image")
		exist, err := exists("templates/images/" + file + ".jpg")
		if err != nil {
			log.Error(err.Error())
		}
		if exist {
			http.ServeFile(w, r, "templates/images/"+file+".jpg")
		} else {
			http.ServeFile(w, r, "templates/images/404.jpg")
		}
	}})

	r.HandleFunc("/key/{key}", PseudoLoginHandler)

	log.Info("listening on %v:%v", s.Config2.Host, s.Config2.Port)
	http.Handle("/", r)
	http.ListenAndServe(s.Config2.Host+":"+s.Config2.Port, nil)
}

//own handler to inject auth before some http handlers
type MyHandler struct {
	F func(http.ResponseWriter, *http.Request)
}

func (m MyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "top-kek")
	if session.Values["login"] != true {
		http.Error(w, "forbidden", 404)
		return
	}
	m.F(w, r)
}

func PseudoLoginHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]

	if key == server.Config2.Key {
		log.Info("Login success from %v", r.RemoteAddr)
		session, _ := store.Get(r, "top-kek")
		session.Values["login"] = true
		session.Save(r, w)
		http.Redirect(w, r, "/", http.StatusFound)
	} else {
		log.Info("Login fail from %v", r.RemoteAddr)
		http.Error(w, "forbidden", http.StatusForbidden)
	}

}

func GetReleaseWithTagAndName(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	offset := vars["offset"]
	tags := vars["tags"]
	name := vars["name"]
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

	fmt.Fprintf(w, string(by))

}

func LinkFollow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	checksum := vars["checksum"]

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

func SetRating(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	checksum := vars["checksum"]
	score, _ := strconv.Atoi(vars["score"])

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

	fmt.Fprintf(w, "%v %v", checksum, score)
}

func GetLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	offset, _ := strconv.Atoi(vars["offset"])

	var all []mydb.Log
	_, err := server.LogDB.Select(&all, "select * from log ORDER BY id DESC LIMIT 50 OFFSET ?", offset)
	if err != nil {
		log.Error(err.Error())
	}
	by, err := json.Marshal(all)
	if err != nil {
		log.Error(err.Error())
	}

	fmt.Fprintf(w, string(by))

}

func GetLogsWithLevel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	offset, _ := strconv.Atoi(vars["offset"])
	level := vars["level"]

	var all []mydb.Log
	_, err := server.LogDB.Select(&all, "select * from log WHERE Lvl = ? ORDER BY id DESC LIMIT 50 OFFSET ?", level, offset)
	if err != nil {
		log.Error(err.Error())
	}

	by, err := json.Marshal(all)
	if err != nil {
		log.Error(err.Error())
	}

	fmt.Fprintf(w, string(by))

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
