package gui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// safeBuf is a goroutine-safe bytes.Buffer for test output capture.
type safeBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *safeBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *safeBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func TestNew(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		mux := http.NewServeMux()
		s := New(mux)
		if s.mux != mux {
			t.Error("mux not set")
		}
		if s.port != 0 {
			t.Errorf("default port = %d, want 0", s.port)
		}
		if s.noBrowser {
			t.Error("noBrowser should default to false")
		}
	})

	t.Run("with options", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		mux := http.NewServeMux()
		s := New(mux,
			WithPort(9876),
			WithNoBrowser(),
			WithStdout(&stdout),
			WithStderr(&stderr),
		)
		if s.port != 9876 {
			t.Errorf("port = %d, want 9876", s.port)
		}
		if !s.noBrowser {
			t.Error("noBrowser should be true")
		}
		if s.stdout != &stdout {
			t.Error("stdout not set")
		}
		if s.stderr != &stderr {
			t.Error("stderr not set")
		}
	})
}

func TestServerRun(t *testing.T) {
	t.Run("starts and serves requests on free port", func(t *testing.T) {
		stdout := &safeBuf{}
		mux := http.NewServeMux()
		mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})

		s := New(mux,
			WithNoBrowser(),
			WithStdout(stdout),
			WithStderr(io.Discard),
		)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		errCh := make(chan error, 1)
		go func() { errCh <- s.Run(ctx) }()

		// Wait for server to start — poll stdout for the URL line.
		var port string
		deadline := time.After(3 * time.Second)
		for {
			select {
			case <-deadline:
				t.Fatal("timeout waiting for server to start")
			case err := <-errCh:
				t.Fatalf("server exited early: %v", err)
			default:
			}
			line := stdout.String()
			if strings.Contains(line, "cure gui: http://127.0.0.1:") {
				// Extract port.
				idx := strings.LastIndex(line, ":")
				port = strings.TrimSpace(line[idx+1:])
				break
			}
			time.Sleep(10 * time.Millisecond)
		}

		// Hit the /health endpoint.
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/health", port))
		if err != nil {
			t.Fatalf("GET /health: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want 200", resp.StatusCode)
		}

		var body map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["status"] != "ok" {
			t.Errorf("body status = %q, want %q", body["status"], "ok")
		}

		// Cancel context — server should shut down.
		cancel()

		select {
		case err := <-errCh:
			if err != nil {
				t.Errorf("Run returned error: %v", err)
			}
		case <-time.After(6 * time.Second):
			t.Fatal("server did not shut down within 6 seconds")
		}
	})

	t.Run("graceful shutdown within deadline", func(t *testing.T) {
		mux := http.NewServeMux()
		s := New(mux,
			WithNoBrowser(),
			WithStdout(io.Discard),
			WithStderr(io.Discard),
		)

		ctx, cancel := context.WithCancel(context.Background())

		errCh := make(chan error, 1)
		go func() { errCh <- s.Run(ctx) }()

		// Give server time to bind.
		time.Sleep(50 * time.Millisecond)

		start := time.Now()
		cancel()

		select {
		case err := <-errCh:
			elapsed := time.Since(start)
			if err != nil {
				t.Errorf("Run returned error: %v", err)
			}
			if elapsed > 5*time.Second {
				t.Errorf("shutdown took %v, should be under 5s", elapsed)
			}
		case <-time.After(6 * time.Second):
			t.Fatal("server did not shut down within 6 seconds")
		}
	})

	t.Run("listen error returns wrapped error", func(t *testing.T) {
		mux := http.NewServeMux()
		// Use a clearly invalid port to force a listen error.
		s := New(mux,
			WithPort(-1),
			WithNoBrowser(),
			WithStdout(io.Discard),
			WithStderr(io.Discard),
		)

		err := s.Run(context.Background())
		if err == nil {
			t.Fatal("expected listen error, got nil")
		}
		if !strings.Contains(err.Error(), "gui: listen") {
			t.Errorf("error = %q, want it to contain %q", err.Error(), "gui: listen")
		}
	})
}

func TestServerRunOutputsURL(t *testing.T) {
	stdout := &safeBuf{}
	mux := http.NewServeMux()
	s := New(mux,
		WithNoBrowser(),
		WithStdout(stdout),
		WithStderr(io.Discard),
	)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- s.Run(ctx) }()

	// Wait for output.
	deadline := time.After(3 * time.Second)
	for {
		select {
		case <-deadline:
			cancel()
			t.Fatal("timeout waiting for URL output")
		case err := <-errCh:
			t.Fatalf("server exited early: %v", err)
		default:
		}
		if strings.Contains(stdout.String(), "cure gui: http://127.0.0.1:") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	<-errCh

	output := stdout.String()
	if !strings.HasPrefix(output, "cure gui: http://127.0.0.1:") {
		t.Errorf("stdout = %q, want prefix %q", output, "cure gui: http://127.0.0.1:")
	}
}
