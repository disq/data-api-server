package server

import (
	"encoding/json"
	"fmt"
	"github.com/alexcesaro/log"
	"gopkg.in/tylerb/graceful.v1"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ServerConfig struct {
	ListenIp      string
	ListenPort    int
	RedisHost     string
	RedisPort     int
	RedisDatabase int
	EventTypes    []EventType
}

type Server struct {
	Config *ServerConfig
	Stats  *Stats
	Logger log.Logger
}

const OK_CONTENT = "Accepted"

func NewServer(c *ServerConfig, s *Stats, l log.Logger) *Server {

	return &Server{
		Config: c,
		Stats:  s,
		Logger: l,
	}
}

func (s *Server) Run() {
	s.Logger.Info("Hello!")
	s.Logger.Infof("Initializing HTTP server on %s:%d...", s.Config.ListenIp, s.Config.ListenPort)

	mux := http.NewServeMux()
	mux.HandleFunc("/crossdomain.xml", poorMansMiddleware(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-type", "application/xml")
		fmt.Fprint(w, `<?xml version="1.0"?>
<!DOCTYPE cross-domain-policy SYSTEM "http://www.macromedia.com/xml/dtds/cross-domain-policy.dtd">
<cross-domain-policy>
   <site-control permitted-cross-domain-policies="all" />
   <allow-http-request-headers-from domain="*" headers="*"/>
   <allow-access-from domain="*" to-ports="*" />
</cross-domain-policy>`)
	}))

	mux.HandleFunc("/v1/", poorMansMiddleware(s.apiHandler))

	mux.HandleFunc("/stats", poorMansMiddleware(s.statsHandler))

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/" {
			http.NotFound(w, req)
			return
		}
		fmt.Fprint(w, "Hello?")
	})

	httpServer := &graceful.Server{
		Timeout: 10 * time.Second,
		Server: &http.Server{
			Addr:    fmt.Sprintf("%s:%d", s.Config.ListenIp, s.Config.ListenPort),
			Handler: mux,
		},
	}

	// Launch in separate goroutine so we can block on the main one
	func() {
		if err := httpServer.ListenAndServe(); err != nil {
			s.Logger.Error(err)
			panic(err)
		}
	}()

	// Wait until server is stopped
	<-httpServer.StopChan()

	s.Logger.Info("Shutting down...")
}

func poorMansMiddleware(fn http.HandlerFunc) http.HandlerFunc {
	return addDefaultHeaders(fn)
}

func addDefaultHeaders(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		fn(w, r)
	}
}

func (s *Server) badRequest(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "400 Bad Request", http.StatusBadRequest)
}

func (s *Server) apiHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Cache-control", "priviate, max-age=0, no-cache")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "-1")

	s.Logger.Debugf("Request from %s: %s", req.RemoteAddr, req.URL.RequestURI())

	pathParts := strings.Split(req.URL.Path, "/")
	if len(pathParts) != 3 {
		s.badRequest(w, req)
		return
	}

	eventName := pathParts[2]

	// Convert multiValues into single values if there's only one element
	multiValues := req.URL.Query()
	values := make(map[string]interface{}, 8) // interface: Array of strings, or (most of the time) a single string

	for k, vArr := range multiValues {
		if len(vArr) == 1 {
			values[k] = vArr[0]
		} else if k == "ts" { // "ts" is always single, get the last one
			values[k] = vArr[len(vArr)-1]
		} else {
			values[k] = vArr
		}
	}

	err := s.handleEvent(&EventRecord{
		name: eventName,
		data: values,
	})
	if err != nil {
		s.badRequest(w, req)
	} else {
		fmt.Fprint(w, OK_CONTENT)
	}
}

func getIntParam(req *http.Request, p string, empty_default, invalid_default int) (val int) {
	s := req.FormValue(p)
	val, err := strconv.Atoi(s)
	if err != nil {
		if s == "" {
			val = empty_default
		} else {
			val = invalid_default
		}
	}
	return
}

func (s *Server) statsHandler(w http.ResponseWriter, req *http.Request) {
	s.Logger.Debugf("Stats request from %s: %s", req.RemoteAddr, req.URL.RequestURI())

	response := make(map[string]interface{})

	defer func() {
		jsonData, _ := json.Marshal(response)
		fmt.Fprintf(w, "%s", string(jsonData))
	}()

	req.ParseForm()
	start := getIntParam(req, "since", 0, -1)
	end := getIntParam(req, "until", 0, -1)
	if start < 0 || end < 0 || (start != 0 && end != 0 && end < start) {
		response["error"] = "Invalid since or until parameters"
		return
	}

	if start != 0 || end != 0 {
		response["since"] = start
		response["until"] = end
	}

	data := make(map[string]int, 8)
	for _, e := range s.Config.EventTypes {
		if start != 0 || end != 0 {
			t, _ := s.Stats.GetCounts(e.Name, start, end)
			data[e.Name] = t
		} else {
			t, _ := s.Stats.GetTotal(e.Name)
			data[e.Name] = t
		}
	}
	response["stats"] = data

}
