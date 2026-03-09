package trace

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/mrlm-net/cure/pkg/terminal"
	"github.com/mrlm-net/cure/pkg/tracer/dns"
	"github.com/mrlm-net/cure/pkg/tracer/event"
	"github.com/mrlm-net/cure/pkg/tracer/formatter"
)

// DNSCommand implements the "cure trace dns" subcommand.
type DNSCommand struct {
	format   string
	outFile  string
	dryRun   bool
	timeout  int
	server   string
	count    int
	interval int
}

func (c *DNSCommand) Name() string        { return "dns" }
func (c *DNSCommand) Description() string { return "Trace DNS resolution for a hostname" }
func (c *DNSCommand) Usage() string {
	return `Usage: cure trace dns <hostname> [options]

Resolves a hostname and emits structured trace events including all returned
IP addresses, CNAME chain, resolution time, and RFC 1918 private IP classification.

Examples:
  cure trace dns example.com
  cure trace dns --server 168.63.129.16 myservice.privatelink.blob.core.windows.net
  cure trace dns --count 10 --interval 5 myservice.blob.core.windows.net
  cure trace dns --format html --out-file report.html example.com`
}

func (c *DNSCommand) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("trace-dns", flag.ContinueOnError)
	fs.StringVar(&c.format, "format", "json", "Output format (json, html)")
	fs.StringVar(&c.outFile, "out-file", "", "Output file (default: stdout)")
	fs.BoolVar(&c.dryRun, "dry-run", false, "Emit events without I/O")
	fs.IntVar(&c.timeout, "timeout", 0, "Query timeout in seconds (0 = use config default)")
	fs.StringVar(&c.server, "server", "", "DNS resolver address (IP or IP:port, e.g. 168.63.129.16)")
	fs.IntVar(&c.count, "count", 1, "Number of times to repeat the query (0 = run until Ctrl+C)")
	fs.IntVar(&c.interval, "interval", 0, "Seconds to wait between repeated queries (implies --count 0 when count is not set)")
	return fs
}

func (c *DNSCommand) Run(ctx context.Context, tc *terminal.Context) error {
	if len(tc.Args) == 0 {
		return fmt.Errorf("missing hostname argument")
	}
	hostname := tc.Args[0]

	// Validate --count; negative values are rejected.
	if c.count < 0 {
		return fmt.Errorf("--count must be 0 (infinite) or greater, got %d", c.count)
	}
	// When --interval is set without an explicit --count, run indefinitely (like ping).
	count := c.count
	if c.interval > 0 && c.count == 1 {
		count = 0
	}

	// Merge timeout with config
	timeout := c.timeout
	if timeout == 0 && tc.Config != nil {
		timeout = tc.Config.Get("timeout", 30).(int)
	}
	if timeout == 0 {
		timeout = 30
	}

	// Merge format with config
	format := c.format
	if format == "" && tc.Config != nil {
		format = tc.Config.Get("format", "json").(string)
	}

	// Normalize --server (validate IP, default port 53)
	server := ""
	if c.server != "" {
		var err error
		server, err = normalizeServer(c.server)
		if err != nil {
			return err
		}
	}

	// Create emitter
	var em event.Emitter
	var outW io.Writer = tc.Stdout
	if c.outFile != "" {
		f, err := os.Create(c.outFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		outW = f
	}
	switch format {
	case "json":
		em = formatter.NewNDJSONEmitter(outW)
	case "html":
		em = formatter.NewHTMLEmitter(outW)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
	defer em.Close()

	// Build options
	opts := []dns.Option{
		dns.WithEmitter(em),
		dns.WithDryRun(c.dryRun),
		dns.WithTimeout(time.Duration(timeout) * time.Second),
		dns.WithCount(count),
		dns.WithInterval(time.Duration(c.interval) * time.Second),
	}
	if server != "" {
		opts = append(opts, dns.WithServer(server))
	}

	return dns.TraceDNS(ctx, hostname, opts...)
}

// normalizeServer parses and normalises a --server flag value.
// Accepts "IP" (port defaults to 53) or "IP:port".
// Rejects hostnames — only IP addresses are accepted to avoid DNS bootstrapping circularity.
func normalizeServer(s string) (string, error) {
	if strings.Contains(s, ":") {
		host, port, err := net.SplitHostPort(s)
		if err != nil {
			return "", fmt.Errorf("invalid --server %q: %w", s, err)
		}
		if net.ParseIP(host) == nil {
			return "", fmt.Errorf("--server must be an IP address, got hostname %q", host)
		}
		return net.JoinHostPort(host, port), nil
	}
	if net.ParseIP(s) == nil {
		return "", fmt.Errorf("--server must be an IP address, got %q", s)
	}
	return net.JoinHostPort(s, "53"), nil
}
