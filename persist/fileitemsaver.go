package persist

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileItemSaver struct {
	filePath    string
	imageDir    string
	concurrency int
	buffer      []interface{}
	bufferMu    sync.Mutex
	flushTick   time.Duration
	csvWriter   *csv.Writer
	file        *os.File
	mu          sync.Mutex
}

type FileSaverOption func(*FileItemSaver)

func WithConcurrency(n int) FileSaverOption {
	return func(s *FileItemSaver) {
		s.concurrency = n
	}
}

func WithFlushInterval(d time.Duration) FileSaverOption {
	return func(s *FileItemSaver) {
		s.flushTick = d
	}
}

func WithImageDir(dir string) FileSaverOption {
	return func(s *FileItemSaver) {
		s.imageDir = dir
	}
}

func NewFileItemSaver(filePath string, opts ...FileSaverOption) (*FileItemSaver, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	f, err := os.Create(absPath)
	if err != nil {
		return nil, err
	}

	s := &FileItemSaver{
		filePath:    absPath,
		imageDir:    "images",
		concurrency: 20,
		buffer:      make([]interface{}, 0, 100),
		flushTick:   5 * time.Second,
		file:        f,
		csvWriter:   csv.NewWriter(f),
	}

	for _, opt := range opts {
		opt(s)
	}

	if err := s.csvWriter.Write([]string{"album_id", "title", "image_count", "items_json", "create_time"}); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *FileItemSaver) GetImageDir() string {
	return s.imageDir
}

func (s *FileItemSaver) GetConcurrency() int {
	return s.concurrency
}

func (s *FileItemSaver) SetConcurrency(n int) {
	if n > 0 {
		s.concurrency = n
	}
}

func (s *FileItemSaver) Out() chan<- interface{} {
	out := make(chan interface{}, s.concurrency*2)
	go s.saveLoop(out)
	return out
}

func (s *FileItemSaver) saveLoop(in <-chan interface{}) {
	ticker := time.NewTicker(s.flushTick)
	defer ticker.Stop()

	for {
		select {
		case item, ok := <-in:
			if !ok {
				s.flush()
				return
			}
			s.bufferMu.Lock()
			s.buffer = append(s.buffer, item)
			shouldFlush := len(s.buffer) >= s.concurrency
			s.bufferMu.Unlock()

			if shouldFlush {
				s.flush()
			}

		case <-ticker.C:
			s.flush()
		}
	}
}

func (s *FileItemSaver) flush() {
	s.bufferMu.Lock()
	if len(s.buffer) == 0 {
		s.bufferMu.Unlock()
		return
	}
	toSave := s.buffer
	s.buffer = make([]interface{}, 0, 100)
	s.bufferMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, item := range toSave {
		if err := s.saveItem(item); err != nil {
			log.Printf("save item error: %v", err)
		}
	}
	s.csvWriter.Flush()
}

func (s *FileItemSaver) saveItem(item interface{}) error {
	switch v := item.(type) {
	case AlbumRecord:
		return s.saveAlbumAsCSV(v)
	case AlbumJSONRecord:
		return s.saveAlbumAsJSON(v)
	default:
		bytes, err := json.Marshal(v)
		if err != nil {
			return err
		}
		record := []string{fmt.Sprintf("%v", v), "", "0", string(bytes), time.Now().Format(time.RFC3339)}
		return s.csvWriter.Write(record)
	}
}

func (s *FileItemSaver) saveAlbumAsCSV(album AlbumRecord) error {
	itemsJSON, err := json.Marshal(album.Items)
	if err != nil {
		itemsJSON = []byte("[]")
	}
	record := []string{
		album.AlbumID,
		album.Title,
		fmt.Sprintf("%d", album.ImageCount),
		string(itemsJSON),
		album.CreateTime.Format(time.RFC3339),
	}
	return s.csvWriter.Write(record)
}

func (s *FileItemSaver) saveAlbumAsJSON(album AlbumJSONRecord) error {
	data, err := json.MarshalIndent(album, "", "  ")
	if err != nil {
		return err
	}
	_, err = s.file.Write(append(data, '\n'))
	return err
}

func (s *FileItemSaver) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.csvWriter != nil {
		s.csvWriter.Flush()
	}
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}

type AlbumRecord struct {
	AlbumID     string        `json:"album_id"`
	Title       string        `json:"title"`
	ImageCount  int           `json:"image_count"`
	Items       []ImageRecord `json:"items"`
	CreateTime  time.Time     `json:"create_time"`
}

type ImageRecord struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	LocalPath string `json:"local_path"`
}

type AlbumJSONRecord struct {
	AlbumID    string        `json:"album_id"`
	Title      string        `json:"title"`
	Items      []ImageRecord `json:"items"`
	CreateTime string        `json:"create_time"`
}
