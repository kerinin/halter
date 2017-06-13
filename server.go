package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

const batchSize = 100

// Server ...
type Server struct {
	dataPath string
	data     *csv.Writer
	rewards  map[string]string
	images   []string
	mu       sync.Mutex
}

// NewServer ...
func NewServer(dataPath, imagesPath string) (server *Server, err error) {
	var data *os.File

	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		log.Printf("Creating %s", dataPath)
		data, err = os.Create(dataPath)
		if err != nil {
			return nil, err
		}
	} else if err == nil {
		log.Printf("Opening %s", dataPath)
		data, err = os.OpenFile(dataPath, os.O_RDWR, 0666)
		if err != nil {
			return nil, err
		}
	}

	rewards := map[string]string{}

	csvReader := csv.NewReader(data)
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(record) < 2 {
			continue
		}
		rewards[record[0]] = record[1]
	}
	log.Printf("Existing rewards: %v", rewards)

	images := []string{}

	err = filepath.Walk(imagesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if _, found := rewards[path]; found {
			return nil
		}
		switch filepath.Ext(info.Name()) {
		case ".jpg", ".png", ".gif":
			images = append(images, path)
		default:
			return nil
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	for i := range images {
		j := rand.Intn(i + 1)
		images[i], images[j] = images[j], images[i]
	}

	return &Server{
		dataPath: dataPath,
		data:     csv.NewWriter(data),
		rewards:  rewards,
		images:   images,
	}, nil
}

// Get ...
func (s *Server) Get(w http.ResponseWriter, r *http.Request) {
	images := make([]string, 0, batchSize)

	s.mu.Lock()
	if len(s.images) < batchSize {
		images = s.images[:]
		s.images = s.images[:0]
	} else {
		images = s.images[:batchSize]
		s.images = s.images[batchSize:]
	}
	s.mu.Unlock()

	enc := json.NewEncoder(w)
	err := enc.Encode(images)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error encoding JSON: %s", err)
		return
	}
}

// Post ...
func (s *Server) Post(w http.ResponseWriter, r *http.Request) {
	var rewards map[string]string
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&rewards)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error decoding JSON: %s", err)
	}

	s.mu.Lock()
	lines := make([][]string, 0, len(rewards))
	for k, v := range rewards {
		if _, found := s.rewards[k]; found {
			continue
		}
		s.rewards[k] = v
		lines = append(lines, []string{k, v})
	}
	s.mu.Unlock()

	if len(lines) == 0 {
		return
	}
	err = s.data.WriteAll(lines)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error writing to file: %s", err)
		return
	}
}
