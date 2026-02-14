package trace

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mrlm-net/cure/pkg/terminal"
	"github.com/mrlm-net/cure/pkg/tracer/event"
	"github.com/mrlm-net/cure/pkg/tracer/formatter"
	"github.com/mrlm-net/cure/pkg/tracer/http"
)

type HTTPCommand struct {
	// Flags
	format  string
	outFile string
	dryRun  bool
	method  string
	data    string
	headers headerFlags
	redact  bool
	timeout int
}

func (c *HTTPCommand) Name() string { return "http" }

func (c *HTTPCommand) Description() string {
	return "Trace HTTP request lifecycle"
}

func (c *HTTPCommand) Usage() string {
	return `Usage: cure trace http <url> [options]

Traces an HTTP request to the specified URL, emitting lifecycle events for
DNS resolution, TCP connection, TLS handshake, request/response.

Examples:
  cure trace http https://example.com
  cure trace http --method POST --data '{"key":"value"}' https://api.example.com
  cure trace http --format html --out-file report.html https://example.com`
}

func (c *HTTPCommand) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("trace-http", flag.ContinueOnError)
	fs.StringVar(&c.format, "format", "json", "Output format (json, html)")
	fs.StringVar(&c.outFile, "out-file", "", "Output file (default: stdout)")
	fs.BoolVar(&c.dryRun, "dry-run", false, "Emit events without I/O")
	fs.StringVar(&c.method, "method", "GET", "HTTP method")
	fs.StringVar(&c.data, "data", "", "Request body")
	fs.Var(&c.headers, "H", "Add header (repeatable)")
	fs.BoolVar(&c.redact, "redact", true, "Redact sensitive headers")
	fs.IntVar(&c.timeout, "timeout", 0, "Request timeout in seconds (0 = use config default)")
	return fs
}

func (c *HTTPCommand) Run(ctx context.Context, tc *terminal.Context) error {
	if len(tc.Args) == 0 {
		return fmt.Errorf("missing URL argument")
	}
	url := tc.Args[0]

	// Merge flags with config (flags take precedence)
	timeout := c.timeout
	if timeout == 0 && tc.Config != nil {
		timeout = tc.Config.Get("timeout", 30).(int)
	}
	format := c.format
	if format == "" && tc.Config != nil {
		format = tc.Config.Get("format", "json").(string)
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

	// Build tracer options
	opts := []http.Option{
		http.WithEmitter(em),
		http.WithDryRun(c.dryRun),
		http.WithMethod(c.method),
		http.WithRedact(c.redact),
	}
	if c.data != "" {
		opts = append(opts, http.WithBodyString(c.data))
	}
	if len(c.headers) > 0 {
		opts = append(opts, http.WithHeaders(c.headers.toMap()))
	}

	// Execute trace
	return http.TraceURL(ctx, url, opts...)
}

// headerFlags is a custom flag type for repeatable -H flags.
type headerFlags []string

func (h *headerFlags) String() string { return "" }

func (h *headerFlags) Set(value string) error {
	*h = append(*h, value)
	return nil
}

func (h headerFlags) toMap() map[string]string {
	m := make(map[string]string, len(h))
	for _, hdr := range h {
		parts := strings.SplitN(hdr, ":", 2)
		if len(parts) == 2 {
			m[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return m
}
