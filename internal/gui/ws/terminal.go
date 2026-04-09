// Package ws provides WebSocket handlers for the GUI server, including
// a PTY-backed terminal emulator.
package ws

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"golang.org/x/net/websocket"
)

// TerminalHandler returns an http.Handler that upgrades to WebSocket and
// bridges a PTY-backed shell session. The terminal runs the user's default
// shell (SHELL env, fallback /bin/sh) in the given working directory.
func TerminalHandler(workDir string) http.Handler {
	return websocket.Handler(func(conn *websocket.Conn) {
		defer conn.Close()

		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}

		cmd := exec.Command(shell)
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(), "TERM=xterm-256color")

		ptmx, err := pty.Start(cmd)
		if err != nil {
			log.Printf("terminal: failed to start PTY: %v", err)
			return
		}
		defer ptmx.Close()

		var wg sync.WaitGroup

		// PTY -> WebSocket
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, 4096)
			for {
				n, err := ptmx.Read(buf)
				if n > 0 {
					if _, werr := conn.Write(buf[:n]); werr != nil {
						return
					}
				}
				if err != nil {
					return
				}
			}
		}()

		// WebSocket -> PTY (handle raw input + JSON control messages)
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, 4096)
			for {
				n, err := conn.Read(buf)
				if err != nil {
					cmd.Process.Signal(os.Interrupt)
					return
				}
				data := buf[:n]

				// Try to parse as JSON control message
				var ctrl struct {
					Type string `json:"type"`
					Cols uint16 `json:"cols"`
					Rows uint16 `json:"rows"`
				}
				if json.Unmarshal(data, &ctrl) == nil && ctrl.Type == "resize" {
					pty.Setsize(ptmx, &pty.Winsize{Cols: ctrl.Cols, Rows: ctrl.Rows})
					continue
				}

				// Raw input -> PTY
				if _, err := ptmx.Write(data); err != nil {
					return
				}
			}
		}()

		// Wait for shell to exit
		cmd.Wait()
		// Send exit message
		exitMsg, _ := json.Marshal(map[string]any{"type": "exit", "code": 0})
		conn.Write(exitMsg)

		// Close connection to unblock goroutines
		conn.Close()
		wg.Wait()
	})
}

// VCSStatusHandler returns an http.Handler for VCS status via WebSocket.
// This is a placeholder for future real-time VCS status updates.
func VCSStatusHandler(workDir string) http.Handler {
	return websocket.Handler(func(conn *websocket.Conn) {
		defer conn.Close()
		io.WriteString(conn, `{"branch":"main","dirty":false}`)
	})
}
