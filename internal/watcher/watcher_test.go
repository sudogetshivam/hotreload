package watcher

import (
	"os"
	"path/filepath"
	"testing"  
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestShouldIgnorePath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"src/main.go", false},
		{"main.go", false},
		{"handler/user.go", false},
		{".git/HEAD", true},
		{"src/.git/config", true},
		{"node_modules/express/index.js", true},
		{"vendor/lib/lib.go", true},
		{"__pycache__/mod.pyc", true},
		{"src/main.go~", true},
		{"src/.#main.go", true},
		{"src/#main.go#", true},
		{"src/._temp", true},
		{"src/main.swp", true},
		{"src/main.swo", true},
		{"bin/server", true},
		{"dist/output.js", true},
		{"build/app", true},
		{".idea/workspace.xml", true},
		{".vscode/settings.json", true},
		{"src/file.tmp", true},
		{"src/file.bak", true},
		{"output.exe", true},
		{"lib.so", true},
		{"lib.dll", true},
		{"compiled.o", true},
		{"archive.a", true},
		{".github/workflows/ci.yml", false},
		{"cmd/server/main.go", false},
		{"internal/handler/routes.go", false},
		{"go.mod", false},
		{"go.sum", false},
	}

	for _, tt := range tests {
		got := shouldIgnorePath(tt.path)
		if got != tt.want {
			t.Errorf("shouldIgnorePath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestShouldIgnoreDir(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{".git", true},
		{"node_modules", true},
		{"vendor", true},
		{"bin", true},
		{"src", false},
		{"cmd", false},
		{"internal", false},
		{".github", false},
		{"pkg", false},
	}

	for _, tt := range tests {
		got := shouldIgnoreDir(tt.name)
		if got != tt.want {
			t.Errorf("shouldIgnoreDir(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIsRelevantEvent(t *testing.T) {
	tests := []struct {
		op   fsnotify.Op
		want bool
	}{
		{fsnotify.Write, true},
		{fsnotify.Create, true},
		{fsnotify.Remove, true},
		{fsnotify.Rename, true},
		{fsnotify.Chmod, false},
	}

	for _, tt := range tests {
		ev := fsnotify.Event{Name: "test.go", Op: tt.op}
		got := isRelevantEvent(ev)
		if got != tt.want {
			t.Errorf("isRelevantEvent(%v) = %v, want %v", tt.op, got, tt.want)
		}
	}
}

func TestWatcherNewDirectory(t *testing.T) {
	dir := t.TempDir()

	w, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	subDir := filepath.Join(dir, "subpkg")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)

	testFile := filepath.Join(subDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package subpkg"), 0644); err != nil {
		t.Fatal(err)
	}

	timeout := time.After(3 * time.Second)
	for {
		select {
		case ev := <-w.Events():
			if ev.Name == testFile {
				return
			}
		case <-timeout:
			t.Fatal("timed out waiting for event from new subdirectory")
		}
	}
}

func TestWatcherIgnoredDirectory(t *testing.T) {
	dir := t.TempDir()

	gitDir := filepath.Join(dir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatal(err)
	}

	w, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	if err := os.WriteFile(filepath.Join(gitDir, "test"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	select {
	case <-w.Events():
		t.Fatal("received event from ignored .git directory")
	case <-time.After(500 * time.Millisecond):
	}
}
