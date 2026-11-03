package watcher

import (
	"context"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors a local OpenAPI file for changes on disk.
type Watcher struct {
	filePath string
	subscribers map[chan struct{}]bool
	mu sync.RWMutex
}

// NewWatcher initializes a file watcher for the specified path.
func NewWatcher(filePath string) *Watcher {
	return &Watcher{
		filePath: filePath,
		subscribers: make(map[chan struct{}]bool),
	}
}

// Subscribe returns a channel that receives a signal whenever the watched file changes.
func (w *Watcher) Subscribe() chan struct{} {
	w.mu.Lock()
	defer w.mu.Unlock()
	ch := make(chan struct{}, 1)
	w.subscribers[ch] = true
	return ch
}

// Unsubscribe removes a channel from the subscriber list.
func (w *Watcher) Unsubscribe(ch chan struct{}) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.subscribers, ch)
	close(ch)
}

func (w *Watcher) notify() {
	w.mu.RLock()
	defer w.mu.RUnlock()
	for ch := range w.subscribers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// Start begins monitoring the file in a background goroutine until context cancellation.
func (w *Watcher) Start(ctx context.Context) error {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	absPath, err := filepath.Abs(w.filePath)
	if err != nil {
		absPath = w.filePath
	}

	dir := filepath.Dir(absPath)
	targetBase := filepath.Base(absPath)

	if err := fsWatcher.Add(dir); err != nil {
		_ = fsWatcher.Close()
		return err
	}

	go func() {
		defer fsWatcher.Close()
		var lastReload time.Time

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-fsWatcher.Events:
				if !ok {
					return
				}
				if filepath.Base(event.Name) == targetBase {
					if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
						// Debounce 250ms
						if time.Since(lastReload) > 250*time.Millisecond {
							lastReload = time.Now()
							log.Printf("[Watcher] Detected change in %s, notifying clients...", targetBase)
							w.notify()
						}
					}
				}
			case err, ok := <-fsWatcher.Errors:
				if !ok {
					return
				}
				log.Printf("[Watcher Error] %v", err)
			}
		}
	}()

	return nil
}
