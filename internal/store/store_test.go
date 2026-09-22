package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStorePersistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "limavm-store-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	s, err := New(tempDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	if err := s.SetLastUsedTemplate("docker"); err != nil {
		t.Fatalf("failed to set template: %v", err)
	}

	added, err := s.ToggleFavorite("my-vm")
	if err != nil || !added {
		t.Fatalf("expected favorite added, got %v, err=%v", added, err)
	}

	if !s.IsFavorite("my-vm") {
		t.Errorf("expected my-vm to be favorite")
	}

	// Verify persistence by reloading from disk
	s2, err := New(tempDir)
	if err != nil {
		t.Fatalf("failed to reload store: %v", err)
	}

	if s2.GetLastUsedTemplate() != "docker" {
		t.Errorf("expected docker, got %s", s2.GetLastUsedTemplate())
	}
	if !s2.IsFavorite("my-vm") {
		t.Errorf("expected favorite my-vm persisted")
	}

	// Check state file exists
	if _, err := os.Stat(filepath.Join(tempDir, "state.json")); err != nil {
		t.Errorf("expected state.json to exist: %v", err)
	}
}
