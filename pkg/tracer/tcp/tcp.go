package tcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/mrlm-net/cure/pkg/tracer/event"
)

// TraceAddr traces a TCP connection to addr (host:port format).
//
// Events emitted:
//   - dns_start, dns_done
//   - tcp_connect_start, tcp_connect_done
//   - tcp_send (if data provided)
//   - tcp_receive
//   - tcp_close
//
// Example:
//
//	err := tcp.TraceAddr(context.Background(), "example.com:443",
//	    tcp.WithEmitter(em),
//	    tcp.WithDataString("GET / HTTP/1.0\r\n\r\n"),
//	)
func TraceAddr(ctx context.Context, addr string, opts ...Option) error {
	cfg := &traceConfig{
		emitter: nil,
		dryRun:  false,
		data:    "",
		timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	traceID := generateTraceID()

	if cfg.dryRun {
		return emitDryRunEvents(cfg.emitter, traceID, addr)
	}

	// Parse host and port
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid address %q: %w", addr, err)
	}

	// DNS resolution
	dnsStart := time.Now()
	if cfg.emitter != nil {
		cfg.emitter.Emit(event.NewEvent("dns_start", traceID, map[string]interface{}{
			"host": host,
		}))
	}

	ips, err := net.DefaultResolver.LookupHost(ctx, host)
	dnsDuration := time.Since(dnsStart).Milliseconds()
	if err != nil {
		if cfg.emitter != nil {
			cfg.emitter.Emit(event.NewEvent("dns_done", traceID, map[string]interface{}{
				"error":       err.Error(),
				"duration_ms": dnsDuration,
			}))
		}
		return fmt.Errorf("DNS lookup failed: %w", err)
	}

	var ip string
	if len(ips) > 0 {
		ip = ips[0]
	}
	if cfg.emitter != nil {
		cfg.emitter.Emit(event.NewEvent("dns_done", traceID, map[string]interface{}{
			"ip":          ip,
			"duration_ms": dnsDuration,
		}))
	}

	// TCP connection
	tcpStart := time.Now()
	if cfg.emitter != nil {
		cfg.emitter.Emit(event.NewEvent("tcp_connect_start", traceID, map[string]interface{}{
			"addr": addr,
		}))
	}

	dialer := &net.Dialer{
		Timeout: cfg.timeout,
	}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	tcpDuration := time.Since(tcpStart).Milliseconds()
	if err != nil {
		if cfg.emitter != nil {
			cfg.emitter.Emit(event.NewEvent("tcp_connect_done", traceID, map[string]interface{}{
				"error":       err.Error(),
				"duration_ms": tcpDuration,
			}))
		}
		return fmt.Errorf("TCP connect failed: %w", err)
	}
	defer conn.Close()

	if cfg.emitter != nil {
		cfg.emitter.Emit(event.NewEvent("tcp_connect_done", traceID, map[string]interface{}{
			"local_addr":  conn.LocalAddr().String(),
			"remote_addr": conn.RemoteAddr().String(),
			"duration_ms": tcpDuration,
		}))
	}

	// Send data if provided
	if cfg.data != "" {
		sendStart := time.Now()
		n, err := conn.Write([]byte(cfg.data))
		sendDuration := time.Since(sendStart).Milliseconds()
		if err != nil {
			if cfg.emitter != nil {
				cfg.emitter.Emit(event.NewEvent("tcp_send", traceID, map[string]interface{}{
					"error":       err.Error(),
					"duration_ms": sendDuration,
				}))
			}
			return fmt.Errorf("TCP send failed: %w", err)
		}
		if cfg.emitter != nil {
			cfg.emitter.Emit(event.NewEvent("tcp_send", traceID, map[string]interface{}{
				"bytes":       n,
				"duration_ms": sendDuration,
			}))
		}

		// Try to receive response
		recvStart := time.Now()
		buf := make([]byte, 4096)
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, err = conn.Read(buf)
		recvDuration := time.Since(recvStart).Milliseconds()
		if err != nil && !errors.Is(err, io.EOF) {
			if cfg.emitter != nil {
				cfg.emitter.Emit(event.NewEvent("tcp_receive", traceID, map[string]interface{}{
					"error":       err.Error(),
					"duration_ms": recvDuration,
				}))
			}
		} else {
			if cfg.emitter != nil {
				cfg.emitter.Emit(event.NewEvent("tcp_receive", traceID, map[string]interface{}{
					"bytes":       n,
					"duration_ms": recvDuration,
				}))
			}
		}
	}

	// Close connection
	if cfg.emitter != nil {
		cfg.emitter.Emit(event.NewEvent("tcp_close", traceID, map[string]interface{}{}))
	}

	return nil
}

// Option is a functional option for TraceAddr.
type Option func(*traceConfig)

type traceConfig struct {
	emitter event.Emitter
	dryRun  bool
	data    string
	timeout time.Duration
}

// WithEmitter sets the event emitter.
func WithEmitter(em event.Emitter) Option {
	return func(cfg *traceConfig) {
		cfg.emitter = em
	}
}

// WithDryRun enables dry-run mode.
func WithDryRun(enabled bool) Option {
	return func(cfg *traceConfig) {
		cfg.dryRun = enabled
	}
}

// WithDataString sets the data to send after connection.
func WithDataString(data string) Option {
	return func(cfg *traceConfig) {
		cfg.data = data
	}
}

// WithTimeout sets the connection timeout. Default: 30s.
func WithTimeout(d time.Duration) Option {
	return func(cfg *traceConfig) {
		cfg.timeout = d
	}
}

func generateTraceID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails.
		return hex.EncodeToString([]byte(fmt.Sprintf("%08x", time.Now().UnixNano())))
	}
	return hex.EncodeToString(b)
}

func emitDryRunEvents(em event.Emitter, traceID, addr string) error {
	if em == nil {
		return nil
	}

	em.Emit(event.NewEvent("dns_start", traceID, map[string]interface{}{"host": "example.com"}))
	em.Emit(event.NewEvent("dns_done", traceID, map[string]interface{}{"ip": "93.184.216.34", "duration_ms": 10}))
	em.Emit(event.NewEvent("tcp_connect_start", traceID, map[string]interface{}{"addr": addr}))
	em.Emit(event.NewEvent("tcp_connect_done", traceID, map[string]interface{}{"local_addr": "127.0.0.1:12345", "remote_addr": addr, "duration_ms": 50}))
	em.Emit(event.NewEvent("tcp_send", traceID, map[string]interface{}{"bytes": 100, "duration_ms": 5}))
	em.Emit(event.NewEvent("tcp_receive", traceID, map[string]interface{}{"bytes": 200, "duration_ms": 10}))
	em.Emit(event.NewEvent("tcp_close", traceID, map[string]interface{}{}))

	return nil
}
