// Package daemon implements a long-lived background process that serves
// git-hint suggestions over a Unix domain socket, avoiding the cost of
// spawning a new process on every keystroke. It shuts down automatically
// after a period of inactivity.
package daemon

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"git-hint/app"
	"git-hint/core"
	"git-hint/engine/history"
	"git-hint/scraper"
)

// idleTTL is how long the daemon stays alive without receiving any
// request before shutting itself down. Every request resets this timer.
const idleTTL = 5 * time.Minute

func socketPath() string {
	user := os.Getenv("USER")
	return fmt.Sprintf("/tmp/githint-%s.sock", user)
}

func lockFilePath() string {
	return socketPath() + ".lock"
}

// Run starts the daemon and blocks until it shuts down, either from
// inactivity, a termination signal, or an unrecoverable socket error.
func Run() {
	// Singleton guard: if another daemon is already running for this
	// user, exit quietly rather than fighting over the same socket.
	lockFile, err := os.OpenFile(lockFilePath(), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return
	}
	defer lockFile.Close()

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return
	}
	defer func() { _ = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN) }()

	go func() { _ = http.ListenAndServe("localhost:6060", nil) }()

	path := socketPath()
	os.Remove(path) // clean up a stale socket left by a crashed previous run

	listener, err := net.Listen("unix", path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "githint daemon: failed to create socket:", err)
		return
	}
	os.Chmod(path, 0600)

	defer func() {
		listener.Close()
		os.Remove(path)
	}()

	// Warm up top commands from history in background.
	// Only runs on first-ever execution (when data directory is empty).
	go warmTopCommandsIfFirstRun()

	// Idle shutdown: reset on every request. When it fires with no
	// activity, it closes the listener, which makes Accept() below
	// return an error and the main loop exit cleanly.
	idleTimer := time.AfterFunc(idleTTL, func() {
		listener.Close()
	})
	defer idleTimer.Stop()

	// Also shut down cleanly on SIGINT/SIGTERM (e.g. terminal closing),
	// so the socket and lock files don't linger on disk.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signals
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			// Listener was closed (idle timeout or signal) — stop the
			// loop instead of spinning on a dead listener.
			return
		}
		idleTimer.Reset(idleTTL)
		go handleConn(conn)
	}
}

// Kill sends SIGTERM to the running daemon process. It finds the process
// by reading the PID stored in the lock file. Exits with a non-zero status
// if no daemon is running or the signal fails.
func Kill() error {
	data, err := os.ReadFile(lockFilePath())
	if err != nil {
		// Lock file doesn't exist — try killing by socket ownership as fallback.
		return killBySocket()
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return killBySocket()
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("githint: no daemon process found (pid %d)", pid)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("githint: failed to kill daemon (pid %d): %w", pid, err)
	}
	return nil
}

// killBySocket uses fuser/lsof to find and kill the process holding the socket.
func killBySocket() error {
	sock := socketPath()
	if _, err := os.Stat(sock); err != nil {
		return fmt.Errorf("githint: daemon is not running (no socket at %s)", sock)
	}

	// Try fuser first, then lsof.
	if out, err := exec.Command("fuser", sock).Output(); err == nil {
		for _, field := range strings.Fields(string(out)) {
			pid, err := strconv.Atoi(field)
			if err != nil || pid <= 0 {
				continue
			}
			if p, err := os.FindProcess(pid); err == nil {
				_ = p.Signal(syscall.SIGTERM)
				return nil
			}
		}
	}

	if out, err := exec.Command("lsof", "-t", sock).Output(); err == nil {
		for _, field := range strings.Fields(string(out)) {
			pid, err := strconv.Atoi(field)
			if err != nil || pid <= 0 {
				continue
			}
			if p, err := os.FindProcess(pid); err == nil {
				_ = p.Signal(syscall.SIGTERM)
				return nil
			}
		}
	}

	// Last resort: just remove the stale socket so the next daemon start works.
	os.Remove(sock)
	return fmt.Errorf("githint: could not find daemon PID, removed stale socket")
}

// warmTopCommandsIfFirstRun crawls the complete help tree for the top 10 most
// frequently used root commands from shell history. Only runs on the first
// ever execution — detected via a sentinel file (.warmed) inside the data dir.
func warmTopCommandsIfFirstRun() {
	dataDir, err := core.DataDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "githint warmup: could not resolve data dir:", err)
		return
	}

	sentinelPath := dataDir + "/.warmed"
	if _, err := os.Stat(sentinelPath); err == nil {
		// Sentinel exists — warmup already ran before.
		return
	}

	commands, err := history.TopRootCommands(10)
	if err != nil {
		fmt.Fprintln(os.Stderr, "githint warmup: could not read shell history:", err)
		return
	}
	if len(commands) == 0 {
		fmt.Fprintln(os.Stderr, "githint warmup: no commands found in shell history, skipping")
		_ = os.WriteFile(sentinelPath, nil, 0600)
		return
	}

	fmt.Fprintf(os.Stderr, "githint warmup: indexing %d commands from history: %v\n", len(commands), commands)

	for i, cmd := range commands {
		binPath, err := exec.LookPath(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "githint warmup: [%d/%d] skipping %q (not found in PATH)\n", i+1, len(commands), cmd)
			continue
		}
		fmt.Fprintf(os.Stderr, "githint warmup: [%d/%d] indexing %q...\n", i+1, len(commands), cmd)
		if err := scraper.CrawlOne(cmd, binPath, scraper.Options{MaxDepth: 999}); err != nil {
			fmt.Fprintf(os.Stderr, "githint warmup: [%d/%d] %q failed: %v\n", i+1, len(commands), cmd, err)
		} else {
			fmt.Fprintf(os.Stderr, "githint warmup: [%d/%d] %q done\n", i+1, len(commands), cmd)
		}
	}

	// Write sentinel so we don't repeat warmup on the next daemon start.
	if err := os.WriteFile(sentinelPath, nil, 0600); err != nil {
		fmt.Fprintln(os.Stderr, "githint warmup: could not write sentinel file:", err)
	}
	fmt.Fprintln(os.Stderr, "githint warmup: complete")
}

// handleConn serves exactly one request per connection: read the request
// line, dispatch it, write back length-prefixed response, close.
func handleConn(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}
	line = strings.TrimSuffix(line, "\n")

	response := dispatch(line)

	// Length-prefixed framing: the payload itself may contain \n and
	// control bytes (used for suggestion highlighting), so we can't use
	// a delimiter — the client reads exactly N bytes instead.
	fmt.Fprintf(conn, "%d\n", len(response))
	conn.Write([]byte(response))
}

// dispatch parses a request line and routes it to the matching app
// function. This mirrors the argument parsing that main.go does for the
// CLI mode — each entry point (CLI args, socket line) is responsible for
// translating its own input format into plain values.
func dispatch(line string) string {
	fields := strings.Split(line, "\t")
	if len(fields) == 0 {
		return ""
	}

	switch fields[0] {

	case "list":
		// list \t buffer \t selected \t col \t render
		if len(fields) < 5 {
			return ""
		}
		buffer := fields[1]

		selected := -1
		if val, err := strconv.Atoi(strings.TrimSpace(fields[2])); err == nil {
			selected = val
		}

		promptCol := 0
		if val, err := strconv.Atoi(strings.TrimSpace(fields[3])); err == nil {
			promptCol = val
		}

		renderMode := strings.TrimSpace(fields[4])
		if renderMode == "" {
			renderMode = "ohmyzsh"
		}

		return app.List(buffer, selected, promptCol, renderMode)

	case "key":
		// key \t keyName \t selected \t buffer
		if len(fields) < 4 {
			return ""
		}
		keyName := fields[1]

		selected, err := strconv.Atoi(strings.TrimSpace(fields[2]))
		if err != nil {
			return ""
		}
		buffer := fields[3]

		return app.Key(keyName, selected, buffer)

	default:
		return ""
	}
}
