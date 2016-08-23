package server

import (
	"github.com/alexcesaro/log"
)

type ServerConfig struct {
	ListenIp      string
	ListenPort    int
	RedisHost     string
	RedisPort     int
	RedisDatabase int
	DataDir       string
}

type Server struct {
	Config *ServerConfig
	Logger log.Logger
}

func NewServer(c *ServerConfig, l log.Logger) *Server {
	return &Server{
		Config: c,
		Logger: l,
	}
}

func (s *Server) Run() {
	s.Logger.Info("Hello!!!")
}
