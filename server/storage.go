package server

import (
	"encoding/json"
	"fmt"
	"github.com/alexcesaro/log"
	"os"
	"strings"
	"sync"
	"time"
)

type StorageConfig struct {
	DataDir string
}

type Storage struct {
	Config  *StorageConfig
	Logger  log.Logger
	wg      sync.WaitGroup
	records chan *EventRecord
}

const DIRECTORY_FORMAT = "2006/01/02"
const FILE_FORMAT = "15_{event}.tsv"

func NewStorage(c *StorageConfig, l log.Logger) (s *Storage) {

	s = &Storage{
		Config:  c,
		Logger:  l,
		records: make(chan *EventRecord),
	}
	return
}

func (s *Storage) RunInBackground() {
	go func() {
		s.Run()
	}()
}

func (s *Storage) Stop() {
	close(s.records)
	s.wg.Wait()
}

func (s *Storage) Run() {
	s.wg.Add(1)

	var (
		ofName string
		of     *os.File
		err    error
	)

	for r := range s.records {
		dir, filename := s.determineStoragePath(r)
		if filename != ofName { // is another file other than our destination file open?
			if of != nil {
				of.Close()
			}

			s.ensureDir(dir)
			openFlags := os.O_APPEND | os.O_WRONLY
			if _, err := os.Stat(filename); err != nil {
				openFlags |= os.O_CREATE
			}

			of, err = os.OpenFile(filename, openFlags, 0666)
			if err != nil {
				s.Logger.Errorf("Could not open %s: %v", filename, err)
				panic(err)
			}
			ofName = filename
		}

		_, err = of.WriteString(s.recordToStorageFormat(r))
	}

	if of != nil {
		of.Close()
	}
	s.wg.Done()
}

func (s *Storage) Enqueue(r *EventRecord) {
	s.records <- r
}

func (s *Storage) recordToStorageFormat(r *EventRecord) string {
	jsonData, _ := json.Marshal(r.data)
	// tsv file (without csv quoting, careful)
	// timestamp	json data
	return fmt.Sprintf("%d\t%v\n", r.ts, string(jsonData))
}

func (s *Storage) determineStoragePath(r *EventRecord) (dir, fileWithDir string) {
	t := time.Now()
	dirPrefix := t.Format(DIRECTORY_FORMAT)
	dir = fmt.Sprintf("%s/%s", s.Config.DataDir, dirPrefix)

	filename := strings.Replace(t.Format(FILE_FORMAT), "{event}", r.name, -1)
	fileWithDir = fmt.Sprintf("%s/%s", dir, filename)
	return
}

func (s *Storage) ensureDir(dir string) {
	if err := os.MkdirAll(dir, os.ModeDir|os.ModePerm); err != nil {
		s.Logger.Errorf("Could not create %s: %v", dir, err)
		panic(err)
	}
}