package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// State represents persistent user preferences and lightweight state.
type State struct {
	LastUsedTemplate string            `json:"last_used_template"`
	Favorites        []string          `json:"favorites"`
	RecentVMs        []string          `json:"recent_vms"`
	CustomSettings   map[string]string `json:"custom_settings"`
}

// Store manages loading and saving State atomically to disk.
type Store struct {
	mu       sync.RWMutex
	filePath string
	state    State
}

// New creates or loads a Store from the specified directory.
func New(dataDir string) (*Store, error) {
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve user home directory: %w", err)
		}
		dataDir = filepath.Join(home, ".local", "share", "limavm-tui")
	}

	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		return nil, fmt.Errorf("create data directory %q: %w", dataDir, err)
	}

	filePath := filepath.Join(dataDir, "state.json")
	s := &Store{
		filePath: filePath,
		state: State{
			Favorites:      []string{},
			RecentVMs:      []string{},
			CustomSettings: make(map[string]string),
		},
	}

	if err := s.load(); err != nil && !os.IsNotExist(err) {
		// If corrupted, log or continue with defaults rather than hard crash
		_ = err
	}

	return s, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("unmarshal state json: %w", err)
	}

	if state.Favorites == nil {
		state.Favorites = []string{}
	}
	if state.RecentVMs == nil {
		state.RecentVMs = []string{}
	}
	if state.CustomSettings == nil {
		state.CustomSettings = make(map[string]string)
	}

	s.state = state
	return nil
}

// Save writes the current in-memory state to disk atomically.
func (s *Store) Save() error {
	s.mu.RLock()
	data, err := json.MarshalIndent(s.state, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	dir := filepath.Dir(s.filePath)
	tmpFile, err := os.CreateTemp(dir, "state-*.json.tmp")
	if err != nil {
		return fmt.Errorf("create temp state file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write state to temp file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("sync temp state file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp state file: %w", err)
	}

	if err := os.Rename(tmpName, s.filePath); err != nil {
		return fmt.Errorf("atomic rename state file: %w", err)
	}

	return nil
}

// GetLastUsedTemplate returns the last template used to create a VM.
func (s *Store) GetLastUsedTemplate() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.LastUsedTemplate
}

// SetLastUsedTemplate updates the last used template and persists.
func (s *Store) SetLastUsedTemplate(tmpl string) error {
	s.mu.Lock()
	s.state.LastUsedTemplate = tmpl
	s.mu.Unlock()
	return s.Save()
}

// IsFavorite checks if a VM is favorited.
func (s *Store) IsFavorite(vmName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, f := range s.state.Favorites {
		if f == vmName {
			return true
		}
	}
	return false
}

// ToggleFavorite adds or removes a VM from favorites and persists.
func (s *Store) ToggleFavorite(vmName string) (bool, error) {
	s.mu.Lock()
	found := false
	var updated []string
	for _, f := range s.state.Favorites {
		if f == vmName {
			found = true
		} else {
			updated = append(updated, f)
		}
	}
	if !found {
		updated = append(updated, vmName)
	}
	s.state.Favorites = updated
	s.mu.Unlock()

	return !found, s.Save()
}

// AddRecentVM records a recently accessed VM.
func (s *Store) AddRecentVM(vmName string) error {
	s.mu.Lock()
	var updated []string
	updated = append(updated, vmName)
	for _, name := range s.state.RecentVMs {
		if name != vmName && len(updated) < 10 {
			updated = append(updated, name)
		}
	}
	s.state.RecentVMs = updated
	s.mu.Unlock()
	return s.Save()
}

// GetFavorites returns a copy of favorite VM names.
func (s *Store) GetFavorites() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	favs := make([]string, len(s.state.Favorites))
	copy(favs, s.state.Favorites)
	return favs
}
