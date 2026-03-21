package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// ServeStdio starts the server in stdio transport mode. It reads
// newline-delimited JSON-RPC 2.0 messages from s.stdin and writes responses to
// s.stdout. Requests are dispatched sequentially.
//
// ServeStdio returns when ctx is cancelled or s.stdin reaches EOF. A clean EOF
// returns nil; a scanner error or write error returns a wrapped error.
func (s *Server) ServeStdio(ctx context.Context) error {
	return s.serveLoop(ctx, s.stdin, s.stdout)
}

// scanResult carries a scanned line or a terminal error from the scan goroutine.
type scanResult struct {
	line []byte
	err  error // non-nil signals EOF (nil err) or scan error
	eof  bool
}

// serveLoop is the shared stdio dispatch loop. It is separated from ServeStdio
// to allow testing with arbitrary io.Reader/io.Writer pairs without relying on
// os.Stdin/os.Stdout.
//
// Scanning is performed in a dedicated goroutine so that context cancellation
// can interrupt the loop even while blocked waiting for the next line. The scan
// goroutine runs for the lifetime of serveLoop and is cleaned up when r is
// closed (EOF) or returns an error.
func (s *Server) serveLoop(ctx context.Context, r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	// Increase the buffer to handle large tool schemas or rich prompt content.
	const maxBuf = 1024 * 1024 // 1 MiB
	scanner.Buffer(make([]byte, maxBuf), maxBuf)

	lines := make(chan scanResult, 1)
	go func() {
		for scanner.Scan() {
			b := scanner.Bytes()
			cp := make([]byte, len(b))
			copy(cp, b)
			lines <- scanResult{line: cp}
		}
		if err := scanner.Err(); err != nil {
			// bufio.ErrTooLong means a single line exceeded the 1 MiB scanner
			// buffer. Unlike a network handler, we cannot recover the stream
			// position after a bufio.Scanner error — the scanner is permanently
			// broken once Scan() returns false. Returning the error here is
			// intentional: stdio is subprocess-launched, and a message larger
			// than 1 MiB indicates a broken or malicious client. The caller
			// will wrap and surface the error via serveLoop.
			lines <- scanResult{err: err}
		} else {
			lines <- scanResult{eof: true}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case res := <-lines:
			if res.eof {
				return nil // clean EOF
			}
			if res.err != nil {
				return fmt.Errorf("mcp: stdio read: %w", res.err)
			}
			if len(res.line) == 0 {
				continue // skip blank lines
			}

			var req jsonrpcRequest
			if err := json.Unmarshal(res.line, &req); err != nil {
				// Parse error: respond with a standard parse-error message and continue.
				resp := errResponse(nil, codeParseError, "parse error: "+err.Error())
				_ = writeResponse(w, resp)
				continue
			}

			resp := s.handleRequest(ctx, req)
			if resp == nil {
				continue // notification — no response required
			}
			if err := writeResponse(w, resp); err != nil {
				return fmt.Errorf("mcp: write response: %w", err)
			}
		}
	}
}

// writeResponse serialises resp as JSON and writes it to w followed by a newline.
func writeResponse(w io.Writer, resp *jsonrpcResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("mcp: marshal response: %w", err)
	}
	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}
