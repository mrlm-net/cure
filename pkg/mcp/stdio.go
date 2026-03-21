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

// serveLoop is the shared stdio dispatch loop. It is separated from ServeStdio
// to allow testing with arbitrary io.Reader/io.Writer pairs without relying on
// os.Stdin/os.Stdout.
func (s *Server) serveLoop(ctx context.Context, r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	// Increase the buffer to handle large tool schemas or rich prompt content.
	const maxBuf = 1024 * 1024 // 1 MiB
	scanner.Buffer(make([]byte, maxBuf), maxBuf)

	for {
		// Check for cancellation before blocking on the next line.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("mcp: stdio read: %w", err)
			}
			return nil // clean EOF
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue // skip blank lines
		}

		var req jsonrpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
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
