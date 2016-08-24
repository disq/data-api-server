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

	stats := server.NewStats(&server.StatsConfig{redisHost, redisPort, redisDb}, logger)

	// Register Events
	eventNames := []string{
		"session_start",
		"session_end",
		"link_clicked",
	}

	// Here we initialize separate Storage instances for each event type.
	// Since each event type will be stored to its own file, there's no reason not to do it in parallel.

	// Iterate event names and create EventTypes, initialize separate Storage worker for each EventType
	et := make([]server.EventType, len(eventNames))
	for i, n := range eventNames {
		e := server.NewEventType(n)
		e.Storage = server.NewStorage(&server.StorageConfig{
			DataDir: *dataDir,
		}, logger)

		et[i] = e
	}

	// Run Storage workers
	for _, e := range et {
		e.Storage.RunInBackground()
	}

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
	server.NewServer(config, stats, logger).Run()

	// Wait for all storage workers to stop
	for _, e := range et {
		e.Storage.Stop()
	}

	stats.Close()
	logger.Info("Goodbye!")
}
