package cleanup

import (
	"log/slog"
	"os"
	"sync"
)

var logger = slog.Default()

// SetLogger overrides the cleanup logger (useful for CLI configured logging).
func SetLogger(l *slog.Logger) {
	if l != nil {
		logger = l
	}
}

// Tracker tracks files that should be cleaned up on interrupt
type Tracker struct {
	files map[string]struct{}
	mu    sync.Mutex
}

// NewTracker creates a new cleanup tracker
func NewTracker() *Tracker {
	return &Tracker{
		files: make(map[string]struct{}),
	}
}

// Register adds a file path to the cleanup list
func (t *Tracker) Register(path string) {
	if path == "" || path == "-" {
		return // Don't track stdout or empty paths
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.files[path] = struct{}{}
}

// Unregister removes a file path from the cleanup list
func (t *Tracker) Unregister(path string) {
	if path == "" || path == "-" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.files, path)
}

// GetAll returns a copy of all currently registered files
func (t *Tracker) GetAll() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	files := make([]string, 0, len(t.files))
	for path := range t.files {
		files = append(files, path)
	}
	return files
}

// Cleanup removes all registered files
func (t *Tracker) Cleanup() {
	t.mu.Lock()
	files := make([]string, 0, len(t.files))
	for path := range t.files {
		files = append(files, path)
	}
	t.files = make(map[string]struct{}) // Clear the map
	t.mu.Unlock()

	for _, path := range files {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			// Best effort cleanup - errors are non-critical
			logger.Warn("cleanup_failed", "file", path, "error", err)
		}
	}
}
