package gui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

// Server is an embedded HTTP server that serves the cure GUI frontend.
// It binds to 127.0.0.1 only (loopback) and discovers a free port via
// the OS when port is 0.
type Server struct {
	port      int
	noBrowser bool
	mux       *http.ServeMux
	stdout    io.Writer
	stderr    io.Writer
}

// Option configures a Server.
type Option func(*Server)

// WithPort sets a fixed port. 0 (default) uses an OS-assigned free port.
func WithPort(port int) Option {
	return func(s *Server) { s.port = port }
}

// WithNoBrowser disables automatic browser opening.
func WithNoBrowser() Option {
	return func(s *Server) { s.noBrowser = true }
}

// WithStdout sets the writer for informational output.
func WithStdout(w io.Writer) Option {
	return func(s *Server) { s.stdout = w }
}

// WithStderr sets the writer for error output.
func WithStderr(w io.Writer) Option {
	return func(s *Server) { s.stderr = w }
}

// New creates a Server with the given mux and options. The mux should be
// pre-configured with any API routes before passing it in; the SPA handler
// is mounted on "/" during Run.
func New(mux *http.ServeMux, opts ...Option) *Server {
	s := &Server{
		mux:    mux,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Run starts listening and serving. It blocks until ctx is cancelled, then
// shuts down gracefully within 5 seconds. The live listener is passed
// directly to http.Serve to avoid a free-port race condition.
func (s *Server) Run(ctx context.Context) error {
	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("gui: listen %s: %w", addr, err)
	}

	port := ln.Addr().(*net.TCPAddr).Port

	// Mount SPA handler with port injection.
	spaHandler := NewSPAHandler(distFS, port)
	s.mux.Handle("/", spaHandler)

	httpSrv := &http.Server{Handler: s.mux}

	fmt.Fprintf(s.stdout, "cure gui: http://127.0.0.1:%d\n", port)

	if !s.noBrowser {
		go OpenBrowser(fmt.Sprintf("http://127.0.0.1:%d", port))
	}

	// Graceful shutdown goroutine.
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutCtx)
	}()

	if err := httpSrv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
