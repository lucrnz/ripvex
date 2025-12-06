package cleanup

import (
	"os"
	"sync"
)

// Tracker tracks files that should be cleaned up on interrupt
type Tracker struct {
	files []string
	mu    sync.Mutex
}

// NewTracker creates a new cleanup tracker
func NewTracker() *Tracker {
	return &Tracker{
		files: make([]string, 0),
	}
}

// Register adds a file path to the cleanup list
func (t *Tracker) Register(path string) {
	if path == "" || path == "-" {
		return // Don't track stdout or empty paths
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.files = append(t.files, path)
}

// Unregister removes a file path from the cleanup list
func (t *Tracker) Unregister(path string) {
	if path == "" || path == "-" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for i, f := range t.files {
		if f == path {
			// Remove by swapping with last element and truncating
			t.files[i] = t.files[len(t.files)-1]
			t.files = t.files[:len(t.files)-1]
			break
		}
	}
}

// GetAll returns a copy of all currently registered files
func (t *Tracker) GetAll() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	files := make([]string, len(t.files))
	copy(files, t.files)
	return files
}

// Cleanup removes all registered files
func (t *Tracker) Cleanup() {
	t.mu.Lock()
	files := make([]string, len(t.files))
	copy(files, t.files)
	t.files = t.files[:0] // Clear the list
	t.mu.Unlock()

	for _, path := range files {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			// Best effort cleanup - errors are non-critical
			_ = err
		}
	}
}
