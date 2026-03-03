package modelmgr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Manifest struct {
	UpdatedAt time.Time                   `json:"updated_at"`
	Installed map[string][]InstalledModel `json:"installed"`
}

type InstalledModel struct {
	ID          string    `json:"id"`
	Version     string    `json:"version"`
	SHA256      string    `json:"sha256"`
	License     string    `json:"license"`
	Attribution string    `json:"attribution"`
	InstalledAt time.Time `json:"installed_at"`
	Path        string    `json:"path"`
}

type ManifestStore struct {
	path string
	mu   sync.RWMutex
}

func NewManifestStore(path string) *ManifestStore {
	return &ManifestStore{path: path}
}

func (s *ManifestStore) Load() (Manifest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var m Manifest
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultManifest(), nil
		}
		return m, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return m, err
	}
	if m.Installed == nil {
		m.Installed = map[string][]InstalledModel{}
	}
	return m, nil
}

func (s *ManifestStore) Save(m Manifest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if m.Installed == nil {
		m.Installed = map[string][]InstalledModel{}
	}
	m.UpdatedAt = time.Now().UTC()

	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0o644)
}

func defaultManifest() Manifest {
	return Manifest{
		UpdatedAt: time.Now().UTC(),
		Installed: map[string][]InstalledModel{},
	}
}
