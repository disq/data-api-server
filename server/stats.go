package server

import (
	"fmt"
	"github.com/alexcesaro/log"
	"github.com/garyburd/redigo/redis"
	"time"
)

type StatsConfig struct {
	Host     string
	Port     int
	Database int
}

type Stats struct {
	*redis.Pool
	Logger log.Logger
}

func NewStats(c *StatsConfig, l log.Logger) *Stats {
	p := &redis.Pool{
		MaxIdle:     10,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, c.Port), redis.DialDatabase(c.Database))
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}

	return &Stats{
		p,
		l,
	}
}

func (s *Stats) getEventKeyFromType(eventName string) string {
	return fmt.Sprintf("eventsByType:%s", eventName)
}

func (s *Stats) CountEvent(e *EventRecord) {
	conn := s.Get()
	defer conn.Close()

	key := s.getEventKeyFromType(e.name)

	// TODO do it in Lua, one less Redis call

	// We could assume that there would be one event for this eventType each nanosecond
	// But we don't and keep a counter to have unique sorted set members
	id, err := conn.Do("INCR", fmt.Sprintf("eventCounter:%s", e.name))
	if err != nil {
		s.Logger.Errorf("INCR failed for %s: %v, skipping stats", e, err)
		return
	}
	// Idea: if we were to store the actual data in Redis as well, we can use a key like <eventType>:<id>.
	// Then we can not just count the data take out time-slices as well (Lua would be a big plus)

	// If ids were generated beforehand (maybe something like <host identifier> + e.tsReceived, or UUID) we can also store the id in Storage to correlate

	// This won't scale at all, O(log(N)) operation
	score := int(e.tsReceived / SECOND_IN_NANOSECONDS) // Second precision
	_, err = conn.Do("ZADD", key, score, id)
	if err != nil {
		s.Logger.Errorf("ZADD failed for %s: %v", key, err)
		return
	}
}

func (s *Stats) GetCounts(eventName string, start, stop int) (count int, err error) {
	conn := s.Get()
	defer conn.Close()

	key := s.getEventKeyFromType(eventName)

	var argStart, argStop interface{}
	if start != 0 {
		argStart = start
	} else {
		argStart = "-Inf"
	}
	if stop != 0 {
		argStop = stop
	} else {
		argStop = "+Inf"
	}
	// Slow as well, O(log(N))
	count, err = redis.Int(conn.Do("ZCOUNT", key, argStart, argStop))
	if err != nil {
		s.Logger.Errorf("ZCOUNT failed for %s(%d,%d): %v", key, start, stop, err)
	}

	return
}

func (s *Stats) GetTotal(eventName string) (count int, err error) {
	conn := s.Get()
	defer conn.Close()

	key := s.getEventKeyFromType(eventName)

	// This is much faster, O(1)
	count, err = redis.Int(conn.Do("ZCARD", key))
	if err != nil {
		s.Logger.Errorf("ZCARD failed for %s: %v", key, err)
	}

	return
}
