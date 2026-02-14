package http

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	nethttp "net/http"
	"net/http/httptrace"
	"strings"
	"time"

	"github.com/mrlm-net/cure/pkg/tracer/event"
)

// TraceURL performs an HTTP request to the specified URL and emits lifecycle events.
//
// Events emitted (in order):
//   - dns_start, dns_done
//   - tcp_connect_start, tcp_connect_done
//   - tls_handshake_start, tls_handshake_done (if HTTPS)
//   - http_request_start, http_response_done
//
// Example:
//
//	em := formatter.NewNDJSONEmitter(os.Stdout)
//	err := http.TraceURL(context.Background(), "https://example.com",
//	    http.WithEmitter(em),
//	    http.WithMethod("POST"),
//	    http.WithBodyString(`{"key":"value"}`),
//	)
func TraceURL(ctx context.Context, url string, opts ...Option) error {
	cfg := &traceConfig{
		emitter: nil,
		dryRun:  false,
		method:  "GET",
		body:    "",
		headers: make(map[string]string),
		redact:  true,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Generate trace ID
	traceID := generateTraceID()

	if cfg.dryRun {
		return emitDryRunEvents(cfg.emitter, traceID, url)
	}

	// Create HTTP request
	var bodyReader io.Reader
	if cfg.body != "" {
		bodyReader = strings.NewReader(cfg.body)
	}

	req, err := nethttp.NewRequestWithContext(ctx, cfg.method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add custom headers
	for k, v := range cfg.headers {
		req.Header.Set(k, v)
	}

	// Set up HTTP trace hooks
	var dnsStart, tcpStart, tlsStart, reqStart time.Time
	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			dnsStart = time.Now()
			if cfg.emitter != nil {
				cfg.emitter.Emit(event.NewEvent("dns_start", traceID, map[string]interface{}{
					"host": info.Host,
				}))
			}
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			duration := time.Since(dnsStart).Milliseconds()
			var ip string
			if len(info.Addrs) > 0 {
				ip = info.Addrs[0].IP.String()
			}
			if cfg.emitter != nil {
				cfg.emitter.Emit(event.NewEvent("dns_done", traceID, map[string]interface{}{
					"ip":          ip,
					"duration_ms": duration,
				}))
			}
		},
		ConnectStart: func(network, addr string) {
			tcpStart = time.Now()
			if cfg.emitter != nil {
				cfg.emitter.Emit(event.NewEvent("tcp_connect_start", traceID, map[string]interface{}{
					"network": network,
					"addr":    addr,
				}))
			}
		},
		ConnectDone: func(network, addr string, err error) {
			duration := time.Since(tcpStart).Milliseconds()
			data := map[string]interface{}{
				"network":     network,
				"addr":        addr,
				"duration_ms": duration,
			}
			if err != nil {
				data["error"] = err.Error()
			}
			if cfg.emitter != nil {
				cfg.emitter.Emit(event.NewEvent("tcp_connect_done", traceID, data))
			}
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
			if cfg.emitter != nil {
				cfg.emitter.Emit(event.NewEvent("tls_handshake_start", traceID, map[string]interface{}{}))
			}
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			duration := time.Since(tlsStart).Milliseconds()
			data := map[string]interface{}{
				"duration_ms": duration,
				"version":     tlsVersionString(state.Version),
			}
			if err != nil {
				data["error"] = err.Error()
			}
			if cfg.emitter != nil {
				cfg.emitter.Emit(event.NewEvent("tls_handshake_done", traceID, data))
			}
		},
		GotFirstResponseByte: func() {
			// Optional: track time to first byte
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(ctx, trace))

	// Emit request start event
	reqStart = time.Now()
	if cfg.emitter != nil {
		headers := redactHeaders(req.Header, cfg.redact)
		cfg.emitter.Emit(event.NewEvent("http_request_start", traceID, map[string]interface{}{
			"method":  cfg.method,
			"url":     url,
			"headers": headers,
		}))
	}

	// Execute request
	client := &nethttp.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Emit response done event
	duration := time.Since(reqStart).Milliseconds()
	if cfg.emitter != nil {
		headers := redactHeaders(resp.Header, cfg.redact)
		cfg.emitter.Emit(event.NewEvent("http_response_done", traceID, map[string]interface{}{
			"status":      resp.StatusCode,
			"headers":     headers,
			"body_size":   len(body),
			"duration_ms": duration,
		}))
	}

	return nil
}

// Option is a functional option for TraceURL.
type Option func(*traceConfig)

type traceConfig struct {
	emitter event.Emitter
	dryRun  bool
	method  string
	body    string
	headers map[string]string
	redact  bool
}

// WithEmitter sets the event emitter. Default: NDJSON to stdout.
func WithEmitter(em event.Emitter) Option {
	return func(cfg *traceConfig) {
		cfg.emitter = em
	}
}

// WithDryRun enables dry-run mode (emit events without actual I/O).
func WithDryRun(enabled bool) Option {
	return func(cfg *traceConfig) {
		cfg.dryRun = enabled
	}
}

// WithMethod sets the HTTP method. Default: GET.
func WithMethod(method string) Option {
	return func(cfg *traceConfig) {
		cfg.method = method
	}
}

// WithBodyString sets the request body as a string.
func WithBodyString(body string) Option {
	return func(cfg *traceConfig) {
		cfg.body = body
	}
}

// WithHeaders adds custom headers to the request.
// Headers with keys "Authorization", "Cookie", "Set-Cookie" are redacted in events.
func WithHeaders(headers map[string]string) Option {
	return func(cfg *traceConfig) {
		cfg.headers = headers
	}
}

// WithRedact enables/disables header redaction. Default: true.
func WithRedact(enabled bool) Option {
	return func(cfg *traceConfig) {
		cfg.redact = enabled
	}
}

// generateTraceID creates a simple trace ID using crypto/rand.
func generateTraceID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails.
		return hex.EncodeToString([]byte(fmt.Sprintf("%08x", time.Now().UnixNano())))
	}
	return hex.EncodeToString(b)
}

// redactHeaders redacts sensitive headers if redaction is enabled.
func redactHeaders(headers nethttp.Header, redact bool) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range headers {
		if redact && isSensitiveHeader(k) {
			result[k] = "[REDACTED]"
		} else {
			if len(v) == 1 {
				result[k] = v[0]
			} else {
				result[k] = v
			}
		}
	}
	return result
}

// isSensitiveHeader checks if a header is sensitive and should be redacted.
func isSensitiveHeader(name string) bool {
	lower := strings.ToLower(name)
	return lower == "authorization" || lower == "cookie" || lower == "set-cookie"
}

// tlsVersionString converts TLS version to string.
func tlsVersionString(version uint16) string {
	switch version {
	case 0x0300:
		return "SSL 3.0"
	case 0x0301:
		return "TLS 1.0"
	case 0x0302:
		return "TLS 1.1"
	case 0x0303:
		return "TLS 1.2"
	case 0x0304:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("unknown (0x%04x)", version)
	}
}

// emitDryRunEvents emits synthetic events without making an actual HTTP request.
func emitDryRunEvents(em event.Emitter, traceID, url string) error {
	if em == nil {
		return nil
	}

	// DNS events
	em.Emit(event.NewEvent("dns_start", traceID, map[string]interface{}{"host": "example.com"}))
	em.Emit(event.NewEvent("dns_done", traceID, map[string]interface{}{"ip": "93.184.216.34", "duration_ms": 10}))

	// TCP events
	em.Emit(event.NewEvent("tcp_connect_start", traceID, map[string]interface{}{"network": "tcp", "addr": "93.184.216.34:443"}))
	em.Emit(event.NewEvent("tcp_connect_done", traceID, map[string]interface{}{"network": "tcp", "addr": "93.184.216.34:443", "duration_ms": 50}))

	// TLS events (if HTTPS)
	if strings.HasPrefix(url, "https://") {
		em.Emit(event.NewEvent("tls_handshake_start", traceID, map[string]interface{}{}))
		em.Emit(event.NewEvent("tls_handshake_done", traceID, map[string]interface{}{"duration_ms": 100, "version": "TLS 1.3"}))
	}

	// HTTP events
	em.Emit(event.NewEvent("http_request_start", traceID, map[string]interface{}{"method": "GET", "url": url}))
	em.Emit(event.NewEvent("http_response_done", traceID, map[string]interface{}{"status": 200, "body_size": 1256, "duration_ms": 300}))

	return nil
}
