package engine

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings" 
	"sync"
	"time"

	"hotreload/internal/proc"
	"hotreload/internal/watcher"
)

const (
	debounceDelay  = 200 * time.Millisecond
	crashWindow    = 3 * time.Second
	maxCrashes     = 3
	crashBackoff   = 5 * time.Second
	killGrace      = 5 * time.Second
	forceKillGrace = 3 * time.Second
)

type Engine struct {
	root        string
	buildCmd    string
	execCmd     string
	watcher     *watcher.Watcher
	mu          sync.Mutex
	cycleMu     sync.Mutex
	server      *proc.Cmd
	cancelCycle context.CancelFunc
	stopCh      chan struct{}
	stopped     chan struct{}
	crashTimes  []time.Time
}

func New(root, buildCmd, execCmd string) (*Engine, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	w, err := watcher.New(absRoot)
	if err != nil {
		return nil, err
	}

	return &Engine{
		root:     absRoot,
		buildCmd: buildCmd,
		execCmd:  execCmd,
		watcher:  w,
		stopCh:   make(chan struct{}),
		stopped:  make(chan struct{}),
	}, nil
}

func (e *Engine) Run() {
	defer close(e.stopped)
	slog.Info("hotreload started", "root", e.root)

	go e.cycle()

	var debounceTimer *time.Timer
	var debounceCh <-chan time.Time

	for {
		select {
		case <-e.stopCh:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			e.shutdown()
			return

		case ev, ok := <-e.watcher.Events():
			if !ok {
				return
			}
			slog.Debug("change detected", "path", ev.Name, "op", ev.Op.String())
			e.cancelCurrentCycle()
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.NewTimer(debounceDelay)
			debounceCh = debounceTimer.C

		case <-debounceCh:
			debounceCh = nil
			go e.cycle()

		case err, ok := <-e.watcher.Errors():
			if !ok {
				return
			}
			slog.Error("watcher error", "error", err)
		}
	}
}

func (e *Engine) Stop() {
	close(e.stopCh)
	<-e.stopped
}

func (e *Engine) cycle() {
	e.cycleMu.Lock()
	defer e.cycleMu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	e.mu.Lock()
	e.cancelCycle = cancel
	e.mu.Unlock()

	e.stopServer()

	if e.shouldBackOff() {
		slog.Warn("crash loop detected, backing off", "duration", crashBackoff)
		select {
		case <-time.After(crashBackoff):
		case <-ctx.Done():
			return
		}
	}

	slog.Info("building", "cmd", e.buildCmd)
	if err := e.build(ctx); err != nil {
		if ctx.Err() != nil {
			slog.Info("build cancelled")
			return
		}
		slog.Error("build failed", "error", err)
		return
	}

	if ctx.Err() != nil {
		return
	}

	slog.Info("starting server", "cmd", e.execCmd)
	srv := proc.Command(e.execCmd)
	if err := srv.Start(); err != nil {
		slog.Error("server failed to start", "error", err)
		return
	}

	e.mu.Lock()
	e.server = srv
	e.mu.Unlock()

	startTime := time.Now()
	go e.monitorServer(srv, startTime)
}

func (e *Engine) build(ctx context.Context) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", strings.ReplaceAll(e.buildCmd, "/", `\`))
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", e.buildCmd)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (e *Engine) cancelCurrentCycle() {
	e.mu.Lock()
	if e.cancelCycle != nil {
		e.cancelCycle()
	}
	e.mu.Unlock()
}

func (e *Engine) stopServer() {
	e.mu.Lock()
	srv := e.server
	e.server = nil
	e.mu.Unlock()

	if srv == nil {
		return
	}

	slog.Info("stopping server")
	srv.Terminate()

	select {
	case <-srv.Done():
	case <-time.After(killGrace):
	}

	slog.Warn("server did not stop gracefully, force killing")
	srv.Kill()

	select {
	case <-srv.Done():
	case <-time.After(forceKillGrace):
		slog.Error("server did not exit after force kill")
	}

	// Small delay to allow Windows to release the socket
	time.Sleep(500 * time.Millisecond)
}

func (e *Engine) monitorServer(srv *proc.Cmd, startTime time.Time) {
	<-srv.Done()
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.server != srv {
		return
	}
	elapsed := time.Since(startTime)
	if elapsed < crashWindow {
		e.crashTimes = append(e.crashTimes, time.Now())
		slog.Warn("server exited quickly", "after", elapsed)
	} else {
		e.crashTimes = nil
	}
	slog.Info("server process exited")
}

func (e *Engine) shouldBackOff() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	now := time.Now()
	recent := make([]time.Time, 0, len(e.crashTimes))
	for _, t := range e.crashTimes {
		if now.Sub(t) < 30*time.Second {
			recent = append(recent, t)
		}
	}
	e.crashTimes = recent
	return len(recent) >= maxCrashes
}

func (e *Engine) shutdown() {
	slog.Info("shutting down")
	e.cancelCurrentCycle()
	e.cycleMu.Lock()
	e.stopServer()
	e.cycleMu.Unlock()
	e.watcher.Close()
}
