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
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"git-hint/app"
)

// idleTTL is how long the daemon stays alive without receiving any
// request before shutting itself down. Every request resets this timer.
const idleTTL = 5 * time.Minute

func sockPath() string {
	user := os.Getenv("USER")
	return fmt.Sprintf("/tmp/githint-%s.sock", user)
}

func lockPath() string {
	return sockPath() + ".lock"
}

// Run starts the daemon and blocks until it shuts down, either from
// inactivity, a termination signal, or an unrecoverable socket error.
func Run() {
	// Singleton guard: if another daemon is already running for this
	// user, exit quietly rather than fighting over the same socket.
	lf, err := os.OpenFile(lockPath(), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return
	}
	defer lf.Close()

	if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return
	}
	defer syscall.Flock(int(lf.Fd()), syscall.LOCK_UN)

	go http.ListenAndServe("localhost:6060", nil)

	path := sockPath()
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

	// Idle shutdown: reset on every request. When it fires with no
	// activity, it closes the listener, which makes Accept() below
	// return an error and the main loop exit cleanly.
	idleTimer := time.AfterFunc(idleTTL, func() {
		listener.Close()
	})
	defer idleTimer.Stop()

	// Also shut down cleanly on SIGINT/SIGTERM (e.g. terminal closing),
	// so the socket and lock files don't linger on disk.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
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

	resultado := processar(line)

	// Length-prefixed framing: the payload itself may contain \n and
	// control bytes (used for suggestion highlighting), so we can't use
	// a delimiter — the client reads exactly N bytes instead.
	fmt.Fprintf(conn, "%d\n", len(resultado))
	conn.Write([]byte(resultado))
}

// processar parses a request line and routes it to the matching app
// function. This mirrors the argument parsing that main.go does for the
// CLI mode — each entry point (CLI args, socket line) is responsible for
// translating its own input format into plain values.
func processar(line string) string {
	partes := strings.Split(line, "\t")
	if len(partes) == 0 {
		return ""
	}

	switch partes[0] {

	case "list":
		// list \t buffer \t selected \t col \t render
		if len(partes) < 5 {
			return ""
		}
		buffer := partes[1]

		selected := -1
		if val, err := strconv.Atoi(strings.TrimSpace(partes[2])); err == nil {
			selected = val
		}

		promptCol := 0
		if val, err := strconv.Atoi(strings.TrimSpace(partes[3])); err == nil {
			promptCol = val
		}

		renderMode := strings.TrimSpace(partes[4])
		if renderMode == "" {
			renderMode = "ohmyzsh"
		}

		return app.List(buffer, selected, promptCol, renderMode)

	case "key":
		// key \t tecla \t selected \t buffer
		if len(partes) < 4 {
			return ""
		}
		tecla := partes[1]

		selected, err := strconv.Atoi(strings.TrimSpace(partes[2]))
		if err != nil {
			return ""
		}
		buffer := partes[3]

		return app.Key(tecla, selected, buffer)

	default:
		return ""
	}
}
