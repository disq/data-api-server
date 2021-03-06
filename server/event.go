package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
)

type EventType struct {
	Name    string
	Storage *Storage
}

func NewEventType(name string) EventType {
	return EventType{Name: name}
}

type EventRecord struct {
	name       string
	tsReceived int64
	data       map[string]interface{}
}

const (
	ALLOWED_PAST_TIME_IN_SECONDS = 86400
	SECOND_IN_NANOSECONDS        = 1000000000
)

func (s *Server) handleEvent(r *EventRecord) error {
	t := s.getEventType(r)
	if t == nil {
		s.Logger.Debug("Invalid event", r)
		return errors.New("Invalid event")
	}

	s.Logger.Debug("Processing:", r)

	s.extractTimestamp(r)
	s.Logger.Debug("Final form:", r)

	t.Storage.Enqueue(r)

	// TODO This can be launched in separate goroutine, or sent to a buffered channel and processed by a worker pool
	// For now just count the event synchronously
	s.Stats.CountEvent(r)

	return nil
}

func (s *Server) getEventType(r *EventRecord) *EventType {
	for _, t := range s.Config.EventTypes {
		if t.Name == r.name {
			return &t
		}
	}
	return nil
}

func (s *Server) extractTimestamp(r *EventRecord) error {
	currentTsNano := time.Now().UnixNano()
	currentTsSecs := int(currentTsNano / SECOND_IN_NANOSECONDS)

	var ts int = 0
	if r.data["ts"] != nil {
		floatTs, err := strconv.ParseFloat(r.data["ts"].(string), 64) // allow for float input
		if err != nil {
			s.Logger.Debugf("Invalid timestamp %s, will override", r.data["ts"].(string))
		}
		ts = int(floatTs) // just chop it off
	}

	if ts > currentTsSecs { // No future times!
		//s.Logger.Warning("Future timestamp %v, will override", ts)
		ts = currentTsSecs
	} else if ts < currentTsSecs-ALLOWED_PAST_TIME_IN_SECONDS { // Past time beyond ALLOWED_PAST_TIME_IN_SECONDS!
		//s.Logger.Warning("Past timestamp %v, will override", ts)
		ts = currentTsSecs
	}

	r.data["ts"] = ts
	r.tsReceived = currentTsNano

	return nil
}

func (r *EventRecord) String() string {
	jsonData, _ := json.Marshal(r.data)
	return fmt.Sprintf("[%s @ %d] %s", r.name, r.tsReceived, string(jsonData))
}
