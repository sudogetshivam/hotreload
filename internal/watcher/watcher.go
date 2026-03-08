package watcher

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// getMaxWatches returns the system limit for file watchers
func getMaxWatches() (uint64, error) {
	if runtime.GOOS == "linux" {
		// Read /proc/sys/fs/inotify/max_user_watches on Linux
		data, err := os.ReadFile("/proc/sys/fs/inotify/max_user_watches")
		if err != nil {
			return 0, err
		}
		var max uint64
		_, err = fmt.Sscanf(string(data), "%d", &max)
		return max, err
	}
	// On Windows and macOS, the limit is much higher or dynamic
	// Windows: limited by available memory (typically 100k+ handles per process)
	// macOS: default is around 4096 per process but can be increased
	return 65536, nil
}

var ignoredDirs = map[string]bool{
	".git":         true,
	".hg":          true,
	".svn":         true,
	"node_modules": true,
	"vendor":       true,
	".idea":        true,
	".vscode":      true,
	"bin":          true,
	"dist":         true,
	"build":        true,
	"__pycache__":  true,
	"tmp":          true,
	"temp":         true,
	".next":        true,
	".cache":       true,
}

var ignoredExtensions = map[string]bool{
	".swp":   true,
	".swo":   true,
	".swx":   true,
	".tmp":   true,
	".bak":   true,
	".exe":   true,
	".o":     true,
	".a":     true,
	".so":    true,
	".dll":   true,
	".dylib": true,
	".pyc":   true,
	".class": true,
}

type Watcher struct {
	fsw        *fsnotify.Watcher
	root       string
	events     chan fsnotify.Event
	errors     chan error
	done       chan struct{}
	mu         sync.Mutex
	dirs       map[string]bool
	watchCount int
	maxWatches uint64
}

func New(root string) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	maxWatches, _ := getMaxWatches()

	w := &Watcher{
		fsw:        fsw,
		root:       root,
		events:     make(chan fsnotify.Event, 256),
		errors:     make(chan error, 16),
		done:       make(chan struct{}),
		dirs:       make(map[string]bool),
		maxWatches: maxWatches,
	}

	if err := w.addRecursive(root); err != nil {
		fsw.Close()
		return nil, err
	}

	// Log warning if approaching system limits
	if w.maxWatches > 0 && uint64(w.watchCount) > w.maxWatches*80/100 {
		slog.Warn("approaching system file watcher limit", "watched", w.watchCount, "max", w.maxWatches)
	}

	go w.loop()
	return w, nil
}

func (w *Watcher) Events() <-chan fsnotify.Event {
	return w.events
}

func (w *Watcher) Errors() <-chan error {
	return w.errors
}

func (w *Watcher) Close() error {
	close(w.done)
	return w.fsw.Close()
}

func (w *Watcher) loop() {
	for {
		select {
		case <-w.done:
			return
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if shouldIgnorePath(ev.Name) {
				continue
			}
			if ev.Has(fsnotify.Create) {
				w.handleCreate(ev.Name)
			}
			if ev.Has(fsnotify.Remove) || ev.Has(fsnotify.Rename) {
				w.handleRemove(ev.Name)
			}
			if isRelevantEvent(ev) {
				select {
				case w.events <- ev:
				default:
				}
			}
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			select {
			case w.errors <- err:
			default:
			}
		}
	}
}

func isRelevantEvent(ev fsnotify.Event) bool {
	return ev.Has(fsnotify.Write) ||
		ev.Has(fsnotify.Create) ||
		ev.Has(fsnotify.Remove) ||
		ev.Has(fsnotify.Rename)
}

func (w *Watcher) handleCreate(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if info.IsDir() {
		if err := w.addRecursive(path); err != nil {
			slog.Warn("failed to watch new directory", "path", path, "error", err)
		}
	}
}

func (w *Watcher) handleRemove(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.dirs[path] {
		delete(w.dirs, path)
		w.fsw.Remove(path)
		w.watchCount--
	}
}

func (w *Watcher) addRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if shouldIgnoreDir(name) && path != w.root {
				return filepath.SkipDir
			}

			// Check if we're approaching system limits
			w.mu.Lock()
			if w.maxWatches > 0 && uint64(w.watchCount) >= w.maxWatches {
				w.mu.Unlock()
				slog.Error("system file watcher limit reached, skipping directory", "path", path, "limit", w.maxWatches)
				return filepath.SkipDir
			}
			w.dirs[path] = true
			w.watchCount++
			w.mu.Unlock()

			if err := w.fsw.Add(path); err != nil {
				slog.Warn("failed to watch directory", "path", path, "error", err)
				w.mu.Lock()
				w.watchCount--
				w.mu.Unlock()
			}
		}
		return nil
	})
}

func shouldIgnoreDir(name string) bool {
	return ignoredDirs[name]
}

func shouldIgnorePath(path string) bool {
	base := filepath.Base(path)

	if strings.HasPrefix(base, ".#") {
		return true
	}
	if strings.HasSuffix(base, "~") {
		return true
	}
	if len(base) > 1 && strings.HasPrefix(base, "#") && strings.HasSuffix(base, "#") {
		return true
	}
	if strings.HasPrefix(base, "._") {
		return true
	}

	ext := filepath.Ext(base)
	if ignoredExtensions[ext] {
		return true
	}

	normalized := filepath.ToSlash(path)
	parts := strings.Split(normalized, "/")
	for _, part := range parts {
		if ignoredDirs[part] {
			return true
		}
	}

	return false
}
