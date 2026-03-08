package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"
    
	"hotreload/internal/engine"
)

func main() {
	root := flag.String("root", ".", "directory to watch for file changes")
	buildCmd := flag.String("build", "", "build command to run on changes")
	execCmd := flag.String("exec", "", "command to run the built server")
	flag.Parse()

	if *buildCmd == "" || *execCmd == "" {
		slog.Error("both --build and --exec flags are required")
		os.Exit(1)
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	e, err := engine.New(*root, *buildCmd, *execCmd)
	if err != nil {
		slog.Error("failed to initialize", "error", err)
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	go func() {
		<-sigCh
		slog.Info("received interrupt signal")
		e.Stop()
	}()

	e.Run()
}
