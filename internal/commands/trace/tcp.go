package trace

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/mrlm-net/cure/pkg/terminal"
	"github.com/mrlm-net/cure/pkg/tracer/event"
	"github.com/mrlm-net/cure/pkg/tracer/formatter"
	"github.com/mrlm-net/cure/pkg/tracer/tcp"
)

type TCPCommand struct {
	format  string
	outFile string
	dryRun  bool
	data    string
	timeout int
}

func (c *TCPCommand) Name() string { return "tcp" }

func (c *TCPCommand) Description() string {
	return "Trace TCP connection lifecycle"
}

func (c *TCPCommand) Usage() string {
	return `Usage: cure trace tcp <addr> [options]

Traces a TCP connection to addr (host:port format).

Examples:
  cure trace tcp example.com:443
  cure trace tcp --data "GET / HTTP/1.0\r\n\r\n" example.com:80`
}

func (c *TCPCommand) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("trace-tcp", flag.ContinueOnError)
	fs.StringVar(&c.format, "format", "json", "Output format (json, html)")
	fs.StringVar(&c.outFile, "out-file", "", "Output file (default: stdout)")
	fs.BoolVar(&c.dryRun, "dry-run", false, "Emit events without I/O")
	fs.StringVar(&c.data, "data", "", "Data to send after connection")
	fs.IntVar(&c.timeout, "timeout", 0, "Connection timeout in seconds")
	return fs
}

func (c *TCPCommand) Run(ctx context.Context, tc *terminal.Context) error {
	if len(tc.Args) == 0 {
		return fmt.Errorf("missing address argument (host:port)")
	}
	addr := tc.Args[0]

	// Merge flags with config
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
	opts := []tcp.Option{
		tcp.WithEmitter(em),
		tcp.WithDryRun(c.dryRun),
	}
	if c.data != "" {
		opts = append(opts, tcp.WithDataString(c.data))
	}
	if c.timeout > 0 {
		opts = append(opts, tcp.WithTimeout(time.Duration(c.timeout)*time.Second))
	}

	return tcp.TraceAddr(ctx, addr, opts...)
}
