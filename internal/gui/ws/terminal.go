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
// bridges a PTY-backed shell session.
func TerminalHandler(workDir string) http.Handler {
	return websocket.Handler(func(conn *websocket.Conn) {
		conn.PayloadType = websocket.BinaryFrame
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

		// PTY output -> WebSocket
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

		// WebSocket input -> PTY
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, 4096)
			for {
				n, err := conn.Read(buf)
				if err != nil {
					if err != io.EOF {
						cmd.Process.Signal(os.Interrupt)
					}
					return
				}
				data := buf[:n]

				// Try JSON control message (resize)
				var ctrl struct {
					Type string `json:"type"`
					Cols uint16 `json:"cols"`
					Rows uint16 `json:"rows"`
				}
				if json.Unmarshal(data, &ctrl) == nil && ctrl.Type == "resize" {
					pty.Setsize(ptmx, &pty.Winsize{Cols: ctrl.Cols, Rows: ctrl.Rows})
					continue
				}

				ptmx.Write(data)
			}
		}()

		cmd.Wait()
		conn.Close()
		wg.Wait()
	})
}
