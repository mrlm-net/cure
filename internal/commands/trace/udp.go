package trace

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/mrlm-net/cure/pkg/terminal"
	"github.com/mrlm-net/cure/pkg/tracer/event"
	"github.com/mrlm-net/cure/pkg/tracer/formatter"
	"github.com/mrlm-net/cure/pkg/tracer/udp"
)

type UDPCommand struct {
	format     string
	outFile    string
	dryRun     bool
	data       string
	recvBuffer int
}

func (c *UDPCommand) Name() string { return "udp" }

func (c *UDPCommand) Description() string {
	return "Trace UDP datagram exchange"
}

func (c *UDPCommand) Usage() string {
	return `Usage: cure trace udp <addr> [options]

Traces a UDP exchange with addr (host:port format).

Examples:
  cure trace udp 1.1.1.1:53 --data <dns-query-bytes>`
}

func (c *UDPCommand) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("trace-udp", flag.ContinueOnError)
	fs.StringVar(&c.format, "format", "json", "Output format (json, html)")
	fs.StringVar(&c.outFile, "out-file", "", "Output file (default: stdout)")
	fs.BoolVar(&c.dryRun, "dry-run", false, "Emit events without I/O")
	fs.StringVar(&c.data, "data", "", "Data to send")
	fs.IntVar(&c.recvBuffer, "recv-buffer", 4096, "Receive buffer size in bytes")
	return fs
}

func (c *UDPCommand) Run(ctx context.Context, tc *terminal.Context) error {
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

	opts := []udp.Option{
		udp.WithEmitter(em),
		udp.WithDryRun(c.dryRun),
	}
	if c.data != "" {
		opts = append(opts, udp.WithDataString(c.data))
	}
	if c.recvBuffer > 0 {
		opts = append(opts, udp.WithRecvBuffer(c.recvBuffer))
	}

	return udp.TraceAddr(ctx, addr, opts...)
}
