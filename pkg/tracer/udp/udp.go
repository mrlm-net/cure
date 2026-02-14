package udp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"time"

	"github.com/mrlm-net/cure/pkg/tracer/event"
)

// TraceAddr traces a UDP exchange with addr (host:port format).
//
// Events emitted:
//   - dns_start, dns_done
//   - udp_send
//   - udp_receive (if response received)
//
// Example:
//
//	err := udp.TraceAddr(context.Background(), "1.1.1.1:53",
//	    udp.WithEmitter(em),
//	    udp.WithDataString("\x00\x00\x01\x00..."),
//	)
func TraceAddr(ctx context.Context, addr string, opts ...Option) error {
	cfg := &traceConfig{
		emitter:    nil,
		dryRun:     false,
		data:       "",
		recvBuffer: 4096,
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

	// Open UDP connection
	conn, err := net.Dial("udp", addr)
	if err != nil {
		return fmt.Errorf("UDP dial failed: %w", err)
	}
	defer conn.Close()

	// Send data
	if cfg.data != "" {
		sendStart := time.Now()
		n, err := conn.Write([]byte(cfg.data))
		sendDuration := time.Since(sendStart).Milliseconds()
		if err != nil {
			if cfg.emitter != nil {
				cfg.emitter.Emit(event.NewEvent("udp_send", traceID, map[string]interface{}{
					"error":       err.Error(),
					"duration_ms": sendDuration,
				}))
			}
			return fmt.Errorf("UDP send failed: %w", err)
		}
		if cfg.emitter != nil {
			cfg.emitter.Emit(event.NewEvent("udp_send", traceID, map[string]interface{}{
				"bytes":       n,
				"duration_ms": sendDuration,
			}))
		}

		// Try to receive response
		recvStart := time.Now()
		buf := make([]byte, cfg.recvBuffer)
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, err = conn.Read(buf)
		recvDuration := time.Since(recvStart).Milliseconds()
		if err != nil {
			if cfg.emitter != nil {
				cfg.emitter.Emit(event.NewEvent("udp_receive", traceID, map[string]interface{}{
					"error":       err.Error(),
					"duration_ms": recvDuration,
				}))
			}
		} else {
			if cfg.emitter != nil {
				cfg.emitter.Emit(event.NewEvent("udp_receive", traceID, map[string]interface{}{
					"bytes":       n,
					"duration_ms": recvDuration,
				}))
			}
		}
	}

	return nil
}

// Option is a functional option for TraceAddr.
type Option func(*traceConfig)

type traceConfig struct {
	emitter    event.Emitter
	dryRun     bool
	data       string
	recvBuffer int
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

// WithDataString sets the data to send.
func WithDataString(data string) Option {
	return func(cfg *traceConfig) {
		cfg.data = data
	}
}

// WithRecvBuffer sets the receive buffer size. Default: 4096 bytes.
func WithRecvBuffer(size int) Option {
	return func(cfg *traceConfig) {
		cfg.recvBuffer = size
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

	em.Emit(event.NewEvent("dns_start", traceID, map[string]interface{}{"host": "1.1.1.1"}))
	em.Emit(event.NewEvent("dns_done", traceID, map[string]interface{}{"ip": "1.1.1.1", "duration_ms": 10}))
	em.Emit(event.NewEvent("udp_send", traceID, map[string]interface{}{"bytes": 50, "duration_ms": 2}))
	em.Emit(event.NewEvent("udp_receive", traceID, map[string]interface{}{"bytes": 100, "duration_ms": 20}))

	return nil
}
