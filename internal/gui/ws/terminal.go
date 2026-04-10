// Package ws provides WebSocket handlers for the GUI server, including
// a PTY-backed terminal emulator.
package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// TerminalHandler returns an http.Handler that upgrades to WebSocket and
// bridges a PTY-backed shell session.
func TerminalHandler(workDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("terminal: websocket upgrade failed: %v", err)
			return
		}
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
			conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"failed to start shell"}`))
			return
		}
		defer ptmx.Close()

		var wg sync.WaitGroup

		// PTY output -> WebSocket (binary frames)
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, 4096)
			for {
				n, err := ptmx.Read(buf)
				if n > 0 {
					if werr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
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
			for {
				msgType, data, err := conn.ReadMessage()
				if err != nil {
					cmd.Process.Signal(os.Interrupt)
					return
				}

				if msgType == websocket.TextMessage {
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
					// Plain text input
					ptmx.Write(data)
				} else {
					// Binary input
					ptmx.Write(data)
				}
			}
		}()

		cmd.Wait()
		exitMsg, _ := json.Marshal(map[string]any{"type": "exit", "code": 0})
		conn.WriteMessage(websocket.TextMessage, exitMsg)
		conn.Close()
		wg.Wait()
	})
}
