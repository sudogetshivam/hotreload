 hotreload

A CLI tool that watches your Go project and automatically rebuilds + restarts your server on file save. No more manually stopping and restarting your server every time you make a change.

---

## The Problem

Every time you change a line of Go code, you have to:
1. Stop the server
2. Run `go build`
3. Start it again

Multiply that by 50 saves a day — it adds up fast. **hotreload** eliminates all of that.

---

## Usage

**Step 1 — Build the tool:**

```bash
go build -o bin/hotreload ./cmd/hotreload
```

**Step 2 — Run it on your project:**

```bash
./bin/hotreload --root ./myproject --build "go build -o ./bin/server ./cmd/server" --exec "./bin/server"
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--root` | Directory to watch for file changes |
| `--build` | Command to build your project |
| `--exec` | Command to run your built binary |

That's it. Edit your code, save, and the server restarts automatically within ~2 seconds.

---

## Features

**Core**
- Watches all files recursively including subfolders
- Triggers first build immediately on startup — no need to save first
- Rebuilds and restarts automatically on file save
- Debounces rapid saves — won't rebuild 10 times if you save 10 times quickly
- Cancels in-progress builds if a new change comes in

**Bonus**
- Detects newly created folders while the tool is running
- Ignores irrelevant files (`.git/`, `node_modules/`, temp files, build artifacts)
- Crash loop protection — if your server crashes 3 times quickly, waits before retrying
- Force-kills stubborn processes that don't shut down cleanly
- Manages OS file watcher limits so it can run for hours without issues

---

## Tests

```bash
go test -v ./...
```

11 tests covering crash loop detection and file watching behaviour.

---

## Requirements

- Go 1.21 or higher
- Works on Windows, Linux, and macOS
