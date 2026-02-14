// Package tracer provides network tracing capabilities for HTTP, TCP, and UDP protocols.
//
// # Event Model
//
// All tracers emit lifecycle events through the Emitter interface. Events are
// timestamped and include protocol-specific data:
//
//	em := formatter.NewNDJSONEmitter(os.Stdout)
//	http.TraceURL(context.Background(), "https://example.com", http.WithEmitter(em))
//
// # Protocols
//
// HTTP: DNS, TCP connect, TLS handshake, request/response
// TCP: DNS, connect, send, receive, close
// UDP: DNS, send, receive
//
// # Output Formats
//
// NDJSON: Streaming newline-delimited JSON (default)
// HTML: Buffered single-page report (via HTMLEmitter)
package tracer
