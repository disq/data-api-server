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
	name string
	ts   int
	data map[string]interface{}
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
	var ts int = 0
	if e.data["ts"] != nil {
		floatTs, err := strconv.ParseFloat(e.data["ts"].(string), 64) // allow for float input
		if err != nil {
			s.Logger.Warning("Invalid timestamp %v, will override", e.data["ts"])
		}
		ts = int(floatTs) // just chop it off
	}

	currentTs := int(time.Now().Unix())
	if ts > currentTs { // No future times!
		//s.Logger.Warning("Future timestamp %v, will override", ts)
		ts = currentTs
	} else if ts < currentTs-ALLOWED_PAST_TIME_IN_SECONDS { // Past time beyond ALLOWED_PAST_TIME_IN_SECONDS!
		//s.Logger.Warning("Past timestamp %v, will override", ts)
		ts = currentTs
	}

	e.ts = ts
	delete(e.data, "ts") // don't keep the redundant "ts" in data

	return nil
}

func (r *EventRecord) String() string {
	jsonData, _ := json.Marshal(r.data)
	return fmt.Sprintf("[%s @ %d] %v", r.name, r.ts, string(jsonData))
}
