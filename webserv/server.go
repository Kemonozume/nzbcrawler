package webserv

import (
	"./../mydb"
	"./../town"
	"encoding/json"
	"fmt"
	log "github.com/dvirsky/go-pylog/logging"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/howeyc/fsnotify"
	"github.com/robfig/config"
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

type Server struct {
	Config  *config.Config
	Config2 *Conf
	RelDB   *mydb.MyDB
	LogDB   *mydb.MyDB
	Watcher *fsnotify.Watcher
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
var templates *template.Template
var store *sessions.CookieStore

//enables live editing of the templates
func (s *Server) WatchFiles() {
	defer func() {
		if err := recover(); err != nil {
			log.Info("recovered from panic")
			s.WatchFiles()
		}
	}()
	for {
		select {
		case <-s.Watcher.Event:
			templates = template.Must(template.ParseFiles("templates/index.html", "templates/log.html"))
		case err := <-s.Watcher.Error:
			log.Info(err.Error())
		}
	}

}

func (s *Server) Init() {

	s.readConfig()

	store = sessions.NewCookieStore([]byte(s.Config2.Secret))
	store.Options = &sessions.Options{
		Path:   "/",
		Domain: "",
		MaxAge: 86400 * 7,
	}

	templates = template.Must(template.ParseFiles("templates/index.html", "templates/log.html"))

	//watch file changes
	var err_tmp error
	s.Watcher, err_tmp = fsnotify.NewWatcher()
	if err_tmp != nil {
		log.Info(err_tmp.Error())
	}

	go s.WatchFiles()

	err := s.Watcher.Watch("templates")
	if err != nil {
		log.Info(err.Error())
	}

	//start Runner...
	if s.Config2.Crawl {
		runner = &Runner{s}
		go runner.Start()
	}

	server = s
	r := mux.NewRouter()

	//browse
	r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/db/events/{offset:[0-9]+}/{tags}/{name}", GetReleaseWithTagAndName)
	r.HandleFunc("/db/event/{checksum}/hits", SetHit)
	r.HandleFunc("/db/event/{checksum}/{score}", SetRating)

	//logs
	r.HandleFunc("/log", LogHandler)
	r.HandleFunc("/log/{offset:[0-9]+}", GetLogs).Methods("GET")
	r.HandleFunc("/log/{offset:[0-9]+}/{level}", GetLogsWithLevel).Methods("GET")
	r.HandleFunc("/log/clearlogs", ClearLogs).Methods("POST")

	//assets
	r.HandleFunc("/public/{file:.+}", AssetHandler)
	r.HandleFunc("/images/{file:.+}", ImgHandler)

	r.HandleFunc("/key/{key}", PseudoLoginHandler)

	log.Info("listening on %v:%v", s.Config2.Host, s.Config2.Port)
	http.Handle("/", r)
	http.ListenAndServe(s.Config2.Host+":"+s.Config2.Port, nil)
}

//BROWSE
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "top-kek")
	if session.Values["login"] != true {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	err := templates.ExecuteTemplate(w, "index.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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
	session, _ := store.Get(r, "top-kek")
	if session.Values["login"] != true {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
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

	if command != "" {
		res, _ := server.RelDB.Eng.Query(command)
		b := Response2Struct(res)
		by, err := json.Marshal(b)
		if err != nil {
			log.Error("json marshal failed %v", err.Error())
		}

		fmt.Fprintf(w, string(by))
	} else {
		var b []town.Release
		offset2, _ := strconv.Atoi(vars["offset"])
		server.RelDB.Eng.Limit(50, offset2).OrderBy("time DESC").Find(&b)
		by, err := json.Marshal(b)
		if err != nil {
			log.Error("json marshal failed %v", err.Error())
		}

		fmt.Fprintf(w, string(by))
	}

}

func SetHit(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "top-kek")
	if session.Values["login"] != true {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	vars := mux.Vars(r)
	checksum := vars["checksum"]

	var b = town.Release{Checksum: checksum}
	has, _ := server.RelDB.Eng.Get(&b)
	if has {
		log.Info("increasing hits for rel: %v", checksum)
		b.Hits += 1
		server.RelDB.Eng.Update(&b, &town.Release{Checksum: checksum})
	}

	fmt.Fprintf(w, "ok")
}

func SetRating(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "top-kek")
	if session.Values["login"] != true {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	checksum := vars["checksum"]
	score, _ := strconv.Atoi(vars["score"])

	var b = town.Release{Checksum: checksum}
	has, _ := server.RelDB.Eng.Get(&b)
	if has {
		log.Info("changing rating for rel: %v with score: %v", checksum, score)
		if score == -1 {
			b.Rating -= 1
			if b.Rating == 0 {
				b.Rating = -1
			}
		} else {
			b.Rating += 1
			if b.Rating == 0 {
				b.Rating = 1
			}
		}
		server.RelDB.Eng.Update(&b, &town.Release{Checksum: checksum})
	}

	fmt.Fprintf(w, "%v %v", checksum, score)
}

//LOGS
func LogHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "top-kek")
	if session.Values["login"] != true {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	err := templates.ExecuteTemplate(w, "log.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

func GetLogs(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "top-kek")
	if session.Values["login"] != true {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	vars := mux.Vars(r)
	offset, err := strconv.Atoi(vars["offset"])

	var all []mydb.Log

	//server.LogDB.Mutex.Lock()
	server.LogDB.Eng.Limit(50, offset).OrderBy("id DESC").Find(&all)
	//server.LogDB.Mutex.Unlock()

	by, err := json.Marshal(all)
	if err != nil {
		log.Error(err.Error())
	}

	fmt.Fprintf(w, string(by))

}

func GetLogsWithLevel(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "top-kek")
	if session.Values["login"] != true {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	vars := mux.Vars(r)
	offset, err := strconv.Atoi(vars["offset"])
	level := vars["level"]

	var all []mydb.Log

	//server.LogDB.Mutex.Lock()
	server.LogDB.Eng.Limit(50, offset).OrderBy("id DESC").Where("Lvl = ?", level).Find(&all)
	//server.LogDB.Mutex.Unlock()

	by, err := json.Marshal(all)
	if err != nil {
		log.Error(err.Error())
	}

	fmt.Fprintf(w, string(by))

}

func ClearLogs(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "defer resp.Body.Close()")
	if session.Values["login"] != true {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	worked := "worked"
	server.LogDB.Mutex.Lock()
	server.LogDB.Eng.Exec("drop table log")
	if err := server.LogDB.Eng.CreateTables(&mydb.Log{}); err != nil {
		log.Error(err.Error())
		worked = "failed"
	}

	server.LogDB.Eng.Exec("vacuum")
	server.LogDB.Mutex.Unlock()

	fmt.Fprintf(w, worked)
}

//ASSETS
func AssetHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "top-kek")
	if session.Values["login"] != true {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	vars := mux.Vars(r)
	file := vars["file"]
	http.ServeFile(w, r, "templates/assets/"+file)
}

func ImgHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "top-kek")
	if session.Values["login"] != true {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	vars := mux.Vars(r)
	file := vars["file"]
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, public, must-revalidate, proxy-revalidate", 432000))
	w.Header().Add("Content-Type", "image")
	http.ServeFile(w, r, "templates/images/"+file)
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

func Response2Struct(res []map[string][]uint8) []town.Release {
	arel := make([]town.Release, len(res))
	var rel town.Release
	for i, val := range res {
		rel = town.Release{}
		rel.Checksum = string(val["checksum"])
		rel.Tag = string(val["tag"])
		rel.Url = string(val["url"])
		rel.Name = string(val["name"])
		arel[i] = rel
	}
	return arel
}

type CMDBuilder struct {
}

func (c *CMDBuilder) Tokenize(token string) string {
	command := "select * from release where "
	atoken := strings.Split(token, "")
	for {
		pos := c.findEarliest(atoken)
		if pos != -1 {
			if pos == 0 { //special char at the beginning
				command = c.buildCommand(command, atoken[pos])
				atoken = atoken[1:]
			} else { //some tag before special char so get the tag...
				cmd := atoken[0:pos]
				spe := atoken[pos]
				command = c.buildCommand(command, strings.Join(cmd, ""))
				command = c.buildCommand(command, spe)
				atoken = atoken[pos+1:]
			}
		} else { //only tag
			command = c.buildCommand(command, strings.Join(atoken[0:], ""))
			atoken = nil
		}
		if atoken == nil {
			break
		}
	}
	return command
}

func (c *CMDBuilder) buildCommand(command string, consumed string) string {
	if consumed == "" {
		return command
	}
	switch consumed {
	case "(", ")":
		command += " " + consumed + " "
	case "|":
		command += " OR "
	case "&":
		command += " AND "
	default:
		if strings.Contains(consumed, "!") {
			consumed = strings.Replace(consumed, "!", "", -1)
			command += "tag NOT LIKE '%" + consumed + "%'"
		} else {
			command += "tag LIKE '%" + consumed + "%'"
		}
	}
	return command
}

func (c *CMDBuilder) findEarliest(token []string) int {
	for i, val := range token {
		if val == "(" || val == ")" || val == "|" || val == "&" {
			return i
		}
	}
	return -1
}
