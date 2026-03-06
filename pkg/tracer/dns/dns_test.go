package dns

import (
	"context"
	"net"
	"testing"

	"github.com/mrlm-net/cure/pkg/tracer/event"
)

// testEmitter captures emitted events for inspection.
type testEmitter struct{ events []event.Event }

func (e *testEmitter) Emit(ev event.Event) error { e.events = append(e.events, ev); return nil }
func (e *testEmitter) Close() error              { return nil }

func TestIsPrivate(t *testing.T) {
	initOnce.Do(initPrivateRanges)

	tests := []struct {
		name string
		ip   string
		want bool
	}{
		// RFC 1918 — 10.0.0.0/8
		{"10.0.0.0 is private", "10.0.0.0", true},
		{"10.255.255.255 is private", "10.255.255.255", true},
		// RFC 1918 — 172.16.0.0/12
		{"172.16.0.0 is private", "172.16.0.0", true},
		{"172.31.255.255 is private", "172.31.255.255", true},
		// RFC 1918 — 192.168.0.0/16
		{"192.168.0.0 is private", "192.168.0.0", true},
		{"192.168.255.255 is private", "192.168.255.255", true},
		// Loopback
		{"127.0.0.1 is private", "127.0.0.1", true},
		{"::1 is private", "::1", true},
		// IPv6 ULA
		{"fc00::1 is private", "fc00::1", true},
		{"fd00::1 is private", "fd00::1", true},
		// Public IPs
		{"8.8.8.8 is public", "8.8.8.8", false},
		{"1.1.1.1 is public", "1.1.1.1", false},
		{"20.0.0.1 is public", "20.0.0.1", false},
		// IPv4-mapped IPv6 — private
		{"::ffff:10.0.0.1 is private", "::ffff:10.0.0.1", true},
		// IPv4-mapped IPv6 — public
		{"::ffff:8.8.8.8 is public", "::ffff:8.8.8.8", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("net.ParseIP(%q) returned nil", tt.ip)
			}
			got := isPrivate(ip)
			if got != tt.want {
				t.Errorf("isPrivate(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestIpFamily(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want string
	}{
		{"plain IPv4", "192.168.1.1", "ipv4"},
		{"loopback IPv4", "127.0.0.1", "ipv4"},
		{"plain IPv6", "2001:db8::1", "ipv6"},
		{"loopback IPv6", "::1", "ipv6"},
		// IPv4-mapped IPv6 must be reported as "ipv4"
		{"IPv4-mapped IPv6", "::ffff:10.0.0.1", "ipv4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("net.ParseIP(%q) returned nil", tt.ip)
			}
			got := ipFamily(ip)
			if got != tt.want {
				t.Errorf("ipFamily(%q) = %q, want %q", tt.ip, got, tt.want)
			}
		})
	}
}

func TestTraceDNS_DryRun(t *testing.T) {
	tests := []struct {
		name       string
		count      int
		wantEvents int
	}{
		{"count=1 emits 2 events", 1, 2},
		{"count=3 emits 6 events", 3, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			em := &testEmitter{}
			err := TraceDNS(context.Background(), "example.com",
				WithEmitter(em),
				WithDryRun(true),
				WithCount(tt.count),
			)
			if err != nil {
				t.Fatalf("TraceDNS() error = %v", err)
			}
			if len(em.events) != tt.wantEvents {
				t.Errorf("got %d events, want %d", len(em.events), tt.wantEvents)
			}
			// Verify event types alternate: start, done, start, done, ...
			for i, ev := range em.events {
				if ev.TraceID == "" {
					t.Errorf("event[%d] has empty TraceID", i)
				}
				if i%2 == 0 {
					if ev.Type != "dns_query_start" {
						t.Errorf("event[%d] type = %q, want dns_query_start", i, ev.Type)
					}
				} else {
					if ev.Type != "dns_query_done" {
						t.Errorf("event[%d] type = %q, want dns_query_done", i, ev.Type)
					}
				}
			}
		})
	}
}

func TestWithCount_NormalisesToOne(t *testing.T) {
	em := &testEmitter{}
	err := TraceDNS(context.Background(), "example.com",
		WithEmitter(em),
		WithDryRun(true),
		WithCount(0), // should be normalised to 1
	)
	if err != nil {
		t.Fatalf("TraceDNS() error = %v", err)
	}
	// count=0 normalised to 1 → 2 events (1 start + 1 done)
	if len(em.events) != 2 {
		t.Errorf("got %d events, want 2 (count=0 should normalise to 1)", len(em.events))
	}
}

func BenchmarkTraceDNS_DryRun(b *testing.B) {
	em := &testEmitter{}
	for b.Loop() {
		em.events = em.events[:0]
		_ = TraceDNS(context.Background(), "example.com",
			WithEmitter(em),
			WithDryRun(true),
			WithCount(1),
		)
	}
}
