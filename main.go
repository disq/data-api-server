package main

import (
	"flag"
	"github.com/alexcesaro/log/stdlog"
	"github.com/disq/data-api-server/server"
	"os"
	"strconv"
	"strings"
)

func main() {
	dataDir := flag.String("datadir", "/tmp", "Path to data directory")
	listenIp := flag.String("host", "0.0.0.0", "IP to bind to")
	listenPort := flag.Int("port", 8080, "Port to listen to")

	redisInfo := flag.String("redis", "127.0.0.1:6379:0", "Redis <host>:<port>:<db>")

	flag.Parse()
	logger := stdlog.GetFromFlags()

	// Sanitize Params
	if _, err := os.Stat(*dataDir); err != nil {
		logger.Errorf("Error stat %s: %v", *dataDir, err)
		panic(err)
	}
	if *listenPort < 1 || *listenPort > 65535 {
		logger.Errorf("Invalid port %d", *listenPort)
		panic("Invalid port")
	}

	redisParts := strings.Split(*redisInfo, ":")
	if len(redisParts) != 3 {
		logger.Error("Invalid redis flag. Usage: <host>:<port>:<db>")
		panic("Invalid redis flag")
	}
	redisHost := redisParts[0]
	if redisHost == "" {
		logger.Error("Cant specify empty redis host")
		panic("Invalid redis host")
	}
	redisPort, err := strconv.Atoi(redisParts[1])
	if err != nil || redisPort < 1 || redisPort > 65535 {
		logger.Errorf("Invalid redis port %v", redisParts[1])
		panic("Invalid redis port")
	}
	redisDb, err := strconv.Atoi(redisParts[2])
	if err != nil || redisDb < 0 || redisDb > 65535 {
		logger.Errorf("Invalid redis db %v", redisParts[2])
		panic("Invalid redis db")
	}

	// Register Events
	et := []server.EventType{
		server.NewEventType("session_start"),
		server.NewEventType("session_end"),
		server.NewEventType("link_clicked"),
	}

	// Configure & Init storage
	storage := server.NewStorage(&server.StorageConfig{
		DataDir: *dataDir,
	}, logger)
	storage.RunInBackground()

	// Configure Server
	config := &server.ServerConfig{
		ListenIp:      *listenIp,
		ListenPort:    *listenPort,
		RedisHost:     redisHost,
		RedisPort:     redisPort,
		RedisDatabase: redisDb,
		EventTypes:    et,
	}

	// Run
	server.NewServer(config, logger, storage).Run()

	storage.Stop()
}
