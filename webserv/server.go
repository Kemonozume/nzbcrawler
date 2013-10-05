package webserv

import (
	"./../mydb"
	"./../town"
	"encoding/json"
	"fmt"
	log "github.com/dvirsky/go-pylog/logging"
	"github.com/gorilla/mux"
	"github.com/howeyc/fsnotify"
	"github.com/robfig/config"
	"html/template"
	_ "io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type Server struct {
	Config   *config.Config
	Config2  *Conf
	RelDB    *mydb.MyDB
	LogDB    *mydb.MyDB
	StatusDB *mydb.MyDB
	Watcher  *fsnotify.Watcher
}

type Conf struct {
	Host, Port, TownName, TownPassword, GhostName, GhostPassword string
}

var server *Server
var runner *Runner
var Cht chan bool
var Chg chan bool
var templates *template.Template

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
			templates = template.Must(template.ParseFiles("templates/index.html", "templates/status.html", "templates/log.html", "templates/config.html"))
		case err := <-s.Watcher.Error:
			log.Info(err.Error())
		}
	}

}

func (s *Server) Init() {

	Cht = make(chan bool)
	Chg = make(chan bool)

	err := s.readConfig()
	if err != nil {
		log.Info(err.Error())
		return
	}

	templates = template.Must(template.ParseFiles("templates/index.html", "templates/status.html", "templates/log.html", "templates/config.html"))

	//watch file changes
	var err_tmp error
	s.Watcher, err_tmp = fsnotify.NewWatcher()
	if err_tmp != nil {
		log.Info(err_tmp.Error())
	}

	go s.WatchFiles()

	err = s.Watcher.Watch("templates")
	if err != nil {
		log.Info(err.Error())
	}

	//start Runner...
	runner = &Runner{s, Cht, Chg}
	runner.Init()
	go runner.Start()

	server = s
	r := mux.NewRouter()
	//browse
	r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/db/{offset:[0-9]+}/", GetRelease)
	r.HandleFunc("/db/{offset:[0-9]+}/{tags}", GetReleaseWithTag)
	r.HandleFunc("/db/{offset:[0-9]+}/none/none", GetRelease)
	r.HandleFunc("/db/{offset:[0-9]+}/{tags}/{name}", GetReleaseWithTagAndName)

	//status
	r.HandleFunc("/status", StatusHandler).Methods("GET")
	r.HandleFunc("/status/running", GetStatus)

	//logs
	r.HandleFunc("/log", LogHandler)
	r.HandleFunc("/log/{offset:[0-9]+}", GetLogs).Methods("GET")
	r.HandleFunc("/log/{offset:[0-9]+}/{level}", GetLogsWithLevel).Methods("GET")
	r.HandleFunc("/log/clearlogs", ClearLogs).Methods("POST")

	//config
	r.HandleFunc("/config", ConfigHandler).Methods("GET")
	r.HandleFunc("/config", UpdateConfig).Methods("POST")

	//assets
	r.HandleFunc("/public/{file:.+}", AssetHandler)
	r.HandleFunc("/images/{file:.+}", ImgHandler)

	log.Info("listening on %v:%v", s.Config2.Host, s.Config2.Port)
	http.Handle("/", r)
	http.ListenAndServe(s.Config2.Host+":"+s.Config2.Port, nil)
}

//BROWSE
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "index.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func GetRelease(w http.ResponseWriter, r *http.Request) {
	log.Info("getRelease()")
	vars := mux.Vars(r)
	offset, err := strconv.Atoi(vars["offset"])
	log.Info("offset: %d", offset)
	var all []town.Release

	//server.RelDB.Mutex.Lock()
	log.Info("db wat..")
	server.RelDB.Eng.Limit(200, offset).OrderBy("time DESC").Find(&all)
	//server.RelDB.Mutex.Unlock()

	log.Info("length: %d", len(all))
	b, err := json.Marshal(all)
	if err != nil {
		log.Error(err.Error())
	}

	fmt.Fprintf(w, string(b))

}

func GetReleaseWithTag(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	offset := vars["offset"]
	tags := vars["tags"]

	var command string
	if strings.Contains(tags, "name:") {
		name := strings.Replace(tags, "name: ", "", -1)
		command = "select * from release where name LIKE '%" + name + "%'"
	} else {
		cb := &CMDBuilder{}
		command = cb.Tokenize(tags)
		command += " ORDER BY time DESC LIMIT 200 OFFSET " + offset
	}

	//server.RelDB.Mutex.Lock()

	res, err := server.RelDB.Eng.Query(command)
	if err != nil {
		log.Info("search with tags: %v", tags)
	}
	b := Response2Struct(res)

	//server.RelDB.Mutex.Unlock()

	by, err := json.Marshal(b)
	if err != nil {
		log.Error("json marshal failed %v", err.Error())
	}

	fmt.Fprintf(w, string(by))
}

func GetReleaseWithTagAndName(w http.ResponseWriter, r *http.Request) {
	log.Info("release with tag and info")

	vars := mux.Vars(r)
	offset := vars["offset"]
	tags := vars["tags"]
	name := vars["name"]
	log.Info("%s %s", tags, name)
	command := ""
	if tags == "none" && name != "none" {
		log.Info("%s", name)
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
		log.Info(command)
		//server.RelDB.Mutex.Lock()
		res, _ := server.RelDB.Eng.Query(command)
		b := Response2Struct(res)
		//server.RelDB.Mutex.Unlock()
		by, err := json.Marshal(b)
		if err != nil {
			log.Error("json marshal failed %v", err.Error())
		}

		fmt.Fprintf(w, string(by))
	} else {
		log.Info("give him all")
		var b []town.Release
		offset2, _ := strconv.Atoi(vars["offset"])
		//server.RelDB.Mutex.Lock()
		server.RelDB.Eng.Limit(50, offset2).OrderBy("time DESC").Find(&b)
		//server.RelDB.Mutex.Unlock()
		by, err := json.Marshal(b)
		if err != nil {
			log.Error("json marshal failed %v", err.Error())
		}

		fmt.Fprintf(w, string(by))
	}

}

//Status
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "status.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func GetStatus(w http.ResponseWriter, r *http.Request) {
	var all []StatusRunner

	//server.StatusDB.Mutex.Lock()
	server.StatusDB.Eng.Find(&all)
	//server.StatusDB.Mutex.Unlock()

	by, err := json.Marshal(all)
	if err != nil {
		log.Error(err.Error())
	}

	fmt.Fprintf(w, string(by))

}

//LOGS
func LogHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "log.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func GetLogs(w http.ResponseWriter, r *http.Request) {
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

	worked := "worked"
	server.LogDB.Mutex.Lock()
	server.LogDB.Eng.Exec("drop table log")
	if err := server.LogDB.Eng.CreateTables(&mydb.Log{}); err != nil {
		log.Error(err.Error())
		worked = "failed"
	}
	server.LogDB.Mutex.Unlock()

	fmt.Fprintf(w, worked)

}

//CONFIG
func ConfigHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "config.html", server.Config2)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func UpdateConfig(w http.ResponseWriter, r *http.Request) {
	server.Config.AddOption("default", "host", r.FormValue("Host"))
	server.Config.AddOption("default", "port", r.FormValue("Port"))
	server.Config.AddOption("default", "town-name", r.FormValue("TownName"))
	server.Config.AddOption("default", "town-password", r.FormValue("TownPassword"))
	server.Config.AddOption("default", "ghost-name", r.FormValue("GhostName"))
	server.Config.AddOption("default", "ghost-password", r.FormValue("GhostPassword"))
	server.readConfig()
	server.Config.WriteFile("default.ini", 0644, "")

	err := templates.ExecuteTemplate(w, "config.html", server.Config2)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

//ASSETS
func AssetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	file := vars["file"]
	http.ServeFile(w, r, "templates/assets/"+file)
}

func ImgHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	file := vars["file"]
	http.ServeFile(w, r, "templates/images/"+file)
}

func (s *Server) readConfig() (err error) {
	s.Config2 = &Conf{}

	s.Config2.Port, err = s.Config.String("default", "port")
	if err != nil {
		log.Error(err.Error())
		return err
	}
	s.Config2.Host, err = s.Config.String("default", "host")
	if err != nil {
		log.Error(err.Error())
		return err
	}
	s.Config2.TownName, err = s.Config.String("default", "town-name")
	if err != nil {
		log.Error(err.Error())
	}
	s.Config2.TownPassword, err = s.Config.String("default", "town-password")
	if err != nil {
		log.Error(err.Error())
	}
	s.Config2.GhostName, err = s.Config.String("default", "ghost-name")
	if err != nil {
		log.Error(err.Error())
	}
	s.Config2.GhostPassword, err = s.Config.String("default", "ghost-password")
	if err != nil {
		log.Error(err.Error())
	}
	return nil

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
