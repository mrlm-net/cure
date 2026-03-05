package trace

import (
	"github.com/mrlm-net/cure/pkg/terminal"
)

// NewTraceCommand creates the trace command group with http/tcp/udp/dns subcommands.
func NewTraceCommand() terminal.Command {
	router := terminal.New(
		terminal.WithName("trace"),
		terminal.WithDescription("Trace network connections (http, tcp, udp, dns)"),
	)
	router.Register(&HTTPCommand{})
	router.Register(&TCPCommand{})
	router.Register(&UDPCommand{})
	router.Register(&DNSCommand{})
	return router
}
