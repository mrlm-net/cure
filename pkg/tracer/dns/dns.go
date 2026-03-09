package dns

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mrlm-net/cure/pkg/tracer/event"
)

var (
	privateRanges []*net.IPNet
	initOnce      sync.Once
)

func initPrivateRanges() {
	for _, cidr := range []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"::1/128",
		"fc00::/7",
	} {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Sprintf("dns: invalid private CIDR %q: %v", cidr, err))
		}
		privateRanges = append(privateRanges, network)
	}
}

// isPrivate reports whether ip is in a private (RFC 1918, loopback, or ULA) range.
func isPrivate(ip net.IP) bool {
	// Normalise IPv4-mapped IPv6 to plain IPv4 for range matching.
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	for _, r := range privateRanges {
		if r.Contains(ip) {
			return true
		}
	}
	return false
}

// ipFamily returns "ipv4" if ip is an IPv4 address, "ipv6" otherwise.
func ipFamily(ip net.IP) string {
	if ip.To4() != nil {
		return "ipv4"
	}
	return "ipv6"
}

func generateTraceID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails.
		return hex.EncodeToString(fmt.Appendf(nil, "%08x", time.Now().UnixNano()))
	}
	return hex.EncodeToString(b)
}

// Option is a functional option for TraceDNS.
type Option func(*traceConfig)

type traceConfig struct {
	emitter  event.Emitter
	dryRun   bool
	timeout  time.Duration
	server   string        // empty = system default resolver; otherwise "IP:port"
	count    int           // default 1
	interval time.Duration // default 0
}

// WithEmitter sets the event emitter.
func WithEmitter(em event.Emitter) Option {
	return func(cfg *traceConfig) {
		cfg.emitter = em
	}
}

// WithDryRun enables dry-run mode, emitting synthetic events without performing real DNS queries.
func WithDryRun(enabled bool) Option {
	return func(cfg *traceConfig) {
		cfg.dryRun = enabled
	}
}

// WithTimeout sets the per-query timeout. Default: 30s.
func WithTimeout(d time.Duration) Option {
	return func(cfg *traceConfig) {
		cfg.timeout = d
	}
}

// WithServer sets the DNS resolver address in "IP:port" form.
// Pass an empty string to use the system default resolver.
func WithServer(addr string) Option {
	return func(cfg *traceConfig) {
		cfg.server = addr
	}
}

// WithCount sets the number of times to repeat the DNS query.
// n = 0 means run indefinitely until the context is cancelled (useful with WithInterval).
// If n < 0 it is set to 1.
func WithCount(n int) Option {
	return func(cfg *traceConfig) {
		if n < 0 {
			n = 1
		}
		cfg.count = n
	}
}

// WithInterval sets the wait duration between repeated queries.
func WithInterval(d time.Duration) Option {
	return func(cfg *traceConfig) {
		cfg.interval = d
	}
}

// buildResolver constructs a *net.Resolver that dials server over UDP.
func buildResolver(server string) *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "udp", server)
		},
	}
}

// readSystemNameservers parses path (typically /etc/resolv.conf) and returns
// the nameserver addresses in "IP:53" form. Returns nil when the file is
// missing, unreadable, or contains no valid nameserver entries.
func readSystemNameservers(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var servers []string
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(strings.TrimSpace(line))
		if len(f) >= 2 && f[0] == "nameserver" && net.ParseIP(f[1]) != nil {
			servers = append(servers, net.JoinHostPort(f[1], "53"))
		}
	}
	return servers
}

// emitDryRunEvents emits synthetic dns_query_start/dns_query_done event pairs.
// count = 0 loops until ctx is cancelled (mirrors the live-query behaviour).
// Uses a hardcoded Azure Private Link scenario as the dry-run payload.
func emitDryRunEvents(ctx context.Context, em event.Emitter, traceID string, count int) error {
	if em == nil {
		return nil
	}
	for attempt := 1; count == 0 || attempt <= count; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		em.Emit(event.NewEvent("dns_query_start", traceID, map[string]any{
			"hostname": "mystorageaccount.blob.core.windows.net",
			"attempt":  attempt,
			"server":   "168.63.129.16:53",
		}))
		em.Emit(event.NewEvent("dns_query_done", traceID, map[string]any{
			"hostname":    "mystorageaccount.blob.core.windows.net",
			"attempt":     attempt,
			"server":      "168.63.129.16:53",
			"cname":       "mystorageaccount.privatelink.blob.core.windows.net.",
			"duration_ms": int64(12),
			"addrs": []map[string]any{
				{"ip": "10.2.0.5", "family": "ipv4", "private": true},
			},
		}))
	}
	return nil
}

// TraceDNS resolves hostname and emits structured trace events for each attempt.
//
// Events emitted per attempt:
//   - dns_query_start
//   - dns_query_done (with addrs on success, error on failure)
//
// Example:
//
//	err := dns.TraceDNS(ctx, "example.com",
//	    dns.WithEmitter(em),
//	    dns.WithCount(3),
//	    dns.WithInterval(2*time.Second),
//	)
func TraceDNS(ctx context.Context, hostname string, opts ...Option) error {
	initOnce.Do(initPrivateRanges)

	cfg := &traceConfig{
		timeout: 30 * time.Second,
		count:   1,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	traceID := generateTraceID()

	if cfg.dryRun {
		return emitDryRunEvents(ctx, cfg.emitter, traceID, cfg.count)
	}

	var resolver *net.Resolver
	// Detect system nameservers once before the loop so we can include them
	// in every event when no explicit server was configured.
	var nameservers []string
	if cfg.server != "" {
		resolver = buildResolver(cfg.server)
	} else {
		resolver = net.DefaultResolver
		nameservers = readSystemNameservers("/etc/resolv.conf")
	}

	for attempt := 1; cfg.count == 0 || attempt <= cfg.count; attempt++ {
		// Wait between repeated queries (skip wait before first attempt).
		if attempt > 1 && cfg.interval > 0 {
			select {
			case <-time.After(cfg.interval):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		iterCtx, cancel := context.WithTimeout(ctx, cfg.timeout)

		// Emit dns_query_start
		startData := map[string]any{
			"hostname": hostname,
			"attempt":  attempt,
		}
		if cfg.server != "" {
			startData["server"] = cfg.server
		} else if len(nameservers) > 0 {
			startData["nameservers"] = nameservers
		}
		if cfg.emitter != nil {
			cfg.emitter.Emit(event.NewEvent("dns_query_start", traceID, startData))
		}

		start := time.Now()

		cname, cnameErr := resolver.LookupCNAME(iterCtx, hostname)
		ipAddrs, ipErr := resolver.LookupIPAddr(iterCtx, hostname)

		duration := time.Since(start).Milliseconds()
		cancel()

		// Build dns_query_done data
		doneData := map[string]any{
			"hostname":    hostname,
			"attempt":     attempt,
			"duration_ms": duration,
		}
		if cfg.server != "" {
			doneData["server"] = cfg.server
		} else if len(nameservers) > 0 {
			doneData["nameservers"] = nameservers
		}

		if ipErr != nil {
			doneData["error"] = ipErr.Error()
			if cfg.emitter != nil {
				cfg.emitter.Emit(event.NewEvent("dns_query_done", traceID, doneData))
			}
			continue
		}

		// Include CNAME only when it differs from the queried hostname (accounting for trailing dot).
		if cnameErr == nil && cname != hostname && cname != hostname+"." {
			doneData["cname"] = cname
		}

		// Build address list with family and private classification.
		addrs := make([]map[string]any, 0, len(ipAddrs))
		for _, ia := range ipAddrs {
			addrs = append(addrs, map[string]any{
				"ip":      ia.IP.String(),
				"family":  ipFamily(ia.IP),
				"private": isPrivate(ia.IP),
			})
		}
		doneData["addrs"] = addrs

		if cfg.emitter != nil {
			cfg.emitter.Emit(event.NewEvent("dns_query_done", traceID, doneData))
		}
	}

	return nil
}
