package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Entry struct {
	CTID   int       `json:"ctid"`
	Digest string    `json:"digest"`
	Status string    `json:"status"`
	Node   string    `json:"node"`
	Update time.Time `json:"update"`
}

type Store interface {
	Load(ctid int) (Entry, bool, error)
	Save(entry Entry) error
	Remove(ctid int) error
}

type FileStore struct {
	dir string
	mu  sync.Mutex
}

func NewFileStore(dir string) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create state dir: %w", err)
	}
	return &FileStore{dir: dir}, nil
}

func (s *FileStore) path(ctid int) string {
	return filepath.Join(s.dir, fmt.Sprintf("%d.json", ctid))
}

func (s *FileStore) Load(ctid int) (Entry, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var entry Entry
	path := s.path(ctid)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return entry, false, nil
		}
		return entry, false, fmt.Errorf("read state: %w", err)
	}
	if err := json.Unmarshal(data, &entry); err != nil {
		return entry, false, fmt.Errorf("decode state: %w", err)
	}
	return entry, true, nil
}

func (s *FileStore) Save(entry Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry.Update = time.Now().UTC()
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	return os.WriteFile(s.path(entry.CTID), data, 0o644)
}

func (s *FileStore) Remove(ctid int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(s.path(ctid)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove state: %w", err)
	}
	return nil
}
