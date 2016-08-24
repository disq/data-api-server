package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
)

type EventType struct {
	name string
}

func NewEventType(name string) EventType {
	return EventType{name}
}

type EventRecord struct {
	name       string
	tsReceived int64
	data       map[string]interface{}
}

const ALLOWED_PAST_TIME_IN_SECONDS = 86400

func (s *Server) handleEvent(r *EventRecord) error {
	t := s.getEventType(r)
	if t == nil {
		s.Logger.Debug("Invalid event", r)
		return errors.New("Invalid event")
	}

	s.Logger.Debug("Processing:", r)

	s.extractTimestamp(r)
	s.Logger.Debug("Final form:", r)

	s.Storage.Enqueue(r)
	// TODO Update stats

	return nil
}

func (s *Server) getEventType(r *EventRecord) *EventType {
	for _, t := range s.Config.EventTypes {
		if t.name == r.name {
			return &t
		}
	}
	return nil
}

func (s *Server) extractTimestamp(e *EventRecord) error {
	currentTsNano := time.Now().UnixNano()
	currentTsSecs := int(currentTsNano / 1000000000)

	var ts int = 0
	if e.data["ts"] != nil {
		floatTs, err := strconv.ParseFloat(e.data["ts"].(string), 64) // allow for float input
		if err != nil {
			s.Logger.Warning("Invalid timestamp %v, will override", e.data["ts"])
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

	e.data["ts"] = ts
	e.tsReceived = currentTsNano

	return nil
}

func (r *EventRecord) String() string {
	jsonData, _ := json.Marshal(r.data)
	return fmt.Sprintf("[%s @ %d] %v", r.name, r.tsReceived, string(jsonData))
}
