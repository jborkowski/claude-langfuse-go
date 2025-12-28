// Package watcher provides file watching functionality for JSONL files.
package watcher

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches for changes in JSONL conversation files.
type Watcher struct {
	watcher   *fsnotify.Watcher
	rootDir   string
	callback  func(string)
	done      chan struct{}
	closeOnce sync.Once

	// Debouncing
	mu           sync.Mutex
	pendingFiles map[string]time.Time
	debounceMs   time.Duration
}

// New creates a new Watcher.
func New(rootDir string, callback func(string)) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		watcher:      fsWatcher,
		rootDir:      rootDir,
		callback:     callback,
		done:         make(chan struct{}),
		pendingFiles: make(map[string]time.Time),
		debounceMs:   500 * time.Millisecond,
	}

	return w, nil
}

// Start begins watching for file changes.
func (w *Watcher) Start() error {
	// Add all directories recursively
	if err := w.addRecursive(w.rootDir); err != nil {
		return err
	}

	// Start event processing goroutine
	go w.processEvents()

	// Start debounce processor
	go w.processPendingFiles()

	return nil
}

// addRecursive adds a directory and all subdirectories to the watch list.
func (w *Watcher) addRecursive(path string) error {
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			if err := w.watcher.Add(path); err != nil {
				// Ignore errors for individual directories
				return nil
			}
		}
		return nil
	})
}

// processEvents handles fsnotify events.
func (w *Watcher) processEvents() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Handle new directories
			if event.Op&fsnotify.Create == fsnotify.Create {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					w.watcher.Add(event.Name)
					continue
				}
			}

			// Only process JSONL files
			if !strings.HasSuffix(event.Name, ".jsonl") {
				continue
			}

			// Handle write and create events
			if event.Op&fsnotify.Write == fsnotify.Write ||
				event.Op&fsnotify.Create == fsnotify.Create {
				w.mu.Lock()
				w.pendingFiles[event.Name] = time.Now()
				w.mu.Unlock()
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			// Log error but continue
			_ = err

		case <-w.done:
			return
		}
	}
}

// processPendingFiles processes files after debounce period.
func (w *Watcher) processPendingFiles() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.mu.Lock()
			now := time.Now()
			for file, lastChange := range w.pendingFiles {
				if now.Sub(lastChange) >= w.debounceMs {
					delete(w.pendingFiles, file)
					// Process file asynchronously
					go w.callback(file)
				}
			}
			w.mu.Unlock()

		case <-w.done:
			return
		}
	}
}

// Close stops the watcher.
func (w *Watcher) Close() error {
	w.closeOnce.Do(func() {
		close(w.done)
	})
	return w.watcher.Close()
}
