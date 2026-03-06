package generate

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/mrlm-net/cure/pkg/template"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// k8sToleration represents a single Kubernetes toleration entry.
type k8sToleration struct {
	Key      string
	Value    string
	Operator string // Equal or Exists
	Effect   string
}

// K8sJobCommand generates a Kubernetes Job manifest for cure trace dns.
type K8sJobCommand struct {
	hostname     string
	namespace    string
	image        string
	version      string
	count        int
	interval     int
	timeout      int
	server       string
	nodeSelector string // comma-separated key=value pairs
	toleration   string // comma-separated key=value:effect or key:effect specs
	output       string // "" = stdout
}

func (c *K8sJobCommand) Name() string { return "k8s-job" }
func (c *K8sJobCommand) Description() string {
	return "Generate a Kubernetes Job manifest to run cure trace dns in-cluster"
}
func (c *K8sJobCommand) Usage() string {
	return `Usage: cure generate k8s-job [flags]

Generate a Kubernetes Job manifest that runs "cure trace dns" inside a cluster.
Useful for diagnosing in-cluster DNS resolution issues from within the pod network.

Required flags:
  --hostname       Hostname to trace (required)

Optional flags:
  --namespace      Kubernetes namespace (default: default)
  --image          Container image (default: golang:1.25-alpine)
  --cure-version   cure version to install, e.g. v0.5.0 (default: latest)
  --count          Number of DNS queries to run (default: 30)
  --interval       Seconds between queries (default: 10)
  --timeout        Per-query timeout in seconds (default: 30)
  --server         DNS server IP address, e.g. 168.63.129.16 (optional)
  --node-selector  Comma-separated key=value node labels, e.g. agentpool=gpupool (optional)
  --toleration     Comma-separated toleration specs: key=value:effect or key:effect (optional)
  --output         Output file path; empty = stdout (default: "")

Examples:
  # Print manifest to stdout
  cure generate k8s-job --hostname myservice.default.svc.cluster.local

  # Target an AKS node pool (VMSS)
  cure generate k8s-job \
    --hostname myservice.blob.core.windows.net \
    --namespace openai-svc \
    --node-selector agentpool=openaisvc \
    --output job.yaml

  # Target a spot node pool (with taint toleration)
  cure generate k8s-job \
    --hostname myservice.blob.core.windows.net \
    --node-selector "kubernetes.azure.com/agentpool=spotnodes" \
    --toleration "kubernetes.azure.com/scalesetpriority=spot:NoSchedule"

  # Apply directly via kubectl
  cure generate k8s-job --hostname api.example.com | kubectl apply -f -
`
}

func (c *K8sJobCommand) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("k8s-job", flag.ContinueOnError)
	fs.StringVar(&c.hostname, "hostname", "", "Hostname to trace (required)")
	fs.StringVar(&c.namespace, "namespace", "default", "Target Kubernetes namespace")
	fs.StringVar(&c.image, "image", "golang:1.25-alpine", "Container image")
	fs.StringVar(&c.version, "cure-version", "latest", "cure version to install (e.g. v0.5.0 or latest)")
	fs.IntVar(&c.count, "count", 30, "Number of DNS queries to run")
	fs.IntVar(&c.interval, "interval", 10, "Seconds between queries")
	fs.IntVar(&c.timeout, "timeout", 30, "Per-query timeout in seconds")
	fs.StringVar(&c.server, "server", "", "DNS server IP address (optional, e.g. 168.63.129.16)")
	fs.StringVar(&c.nodeSelector, "node-selector", "", "Comma-separated key=value node labels (e.g. agentpool=gpupool)")
	fs.StringVar(&c.toleration, "toleration", "", "Comma-separated toleration specs: key=value:effect or key:effect")
	fs.StringVar(&c.output, "output", "", "Output file path (empty = stdout)")
	return fs
}

func (c *K8sJobCommand) Run(ctx context.Context, tc *terminal.Context) error {
	// Validate required --hostname flag.
	if c.hostname == "" {
		return fmt.Errorf("--hostname is required")
	}

	// Validate --server if provided: must be an IP, not a hostname.
	if c.server != "" {
		if err := validateServerIP(c.server); err != nil {
			return err
		}
	}

	jobName := buildJobName(c.hostname)

	nodeSelector, err := parseNodeSelector(c.nodeSelector)
	if err != nil {
		return err
	}

	tolerations, err := parseTolerations(c.toleration)
	if err != nil {
		return err
	}

	data := map[string]any{
		"JobName":      jobName,
		"Namespace":    c.namespace,
		"Hostname":     c.hostname,
		"Image":        c.image,
		"Version":      c.version,
		"Count":        c.count,
		"Interval":     c.interval,
		"Timeout":      c.timeout,
		"Server":       c.server,
		"NodeSelector": nodeSelector,
		"Tolerations":  tolerations,
	}

	output, err := template.Render("k8s-job", data)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	if c.output == "" {
		_, err = fmt.Fprint(tc.Stdout, output)
		return err
	}

	if err := os.WriteFile(c.output, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", c.output, err)
	}

	fmt.Fprintf(tc.Stdout, "Generated %s\n", c.output)
	fmt.Fprintf(tc.Stdout, "\nApply with:\n  kubectl apply -f %s\n", c.output)
	return nil
}

// parseNodeSelector parses a comma-separated "key=value" string into a map.
// Returns nil (not an empty map) when s is empty so the template can test {{if .NodeSelector}}.
func parseNodeSelector(s string) (map[string]string, error) {
	if s == "" {
		return nil, nil
	}
	result := make(map[string]string)
	for pair := range strings.SplitSeq(s, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		k, v, ok := strings.Cut(pair, "=")
		if !ok || k == "" || v == "" {
			return nil, fmt.Errorf("invalid --node-selector %q: expected key=value", pair)
		}
		result[k] = v
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

// parseTolerations parses a comma-separated toleration spec string.
// Each spec is either "key=value:effect" (operator: Equal) or "key:effect" (operator: Exists).
// Returns nil when s is empty so the template can test {{if .Tolerations}}.
func parseTolerations(s string) ([]k8sToleration, error) {
	if s == "" {
		return nil, nil
	}
	var result []k8sToleration
	for spec := range strings.SplitSeq(s, ",") {
		spec = strings.TrimSpace(spec)
		if spec == "" {
			continue
		}
		t, err := parseSingleToleration(spec)
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, nil
}

// parseSingleToleration parses one toleration spec.
// Formats: "key=value:effect" → Equal operator, "key:effect" → Exists operator.
func parseSingleToleration(spec string) (k8sToleration, error) {
	// Split off the trailing ":effect" first.
	lastColon := strings.LastIndex(spec, ":")
	if lastColon < 0 {
		return k8sToleration{}, fmt.Errorf("invalid --toleration %q: expected key:effect or key=value:effect", spec)
	}
	keyPart := spec[:lastColon]
	effect := spec[lastColon+1:]
	if effect == "" {
		return k8sToleration{}, fmt.Errorf("invalid --toleration %q: effect is empty", spec)
	}

	if k, v, ok := strings.Cut(keyPart, "="); ok {
		// key=value:effect → operator Equal
		if k == "" || v == "" {
			return k8sToleration{}, fmt.Errorf("invalid --toleration %q: key and value must be non-empty", spec)
		}
		return k8sToleration{Key: k, Value: v, Operator: "Equal", Effect: effect}, nil
	}
	// key:effect → operator Exists
	if keyPart == "" {
		return k8sToleration{}, fmt.Errorf("invalid --toleration %q: key is empty", spec)
	}
	return k8sToleration{Key: keyPart, Operator: "Exists", Effect: effect}, nil
}

// validateServerIP ensures the server value is an IP address (with optional port),
// not a hostname. Mirrors the normalizeServer logic in internal/commands/trace/dns.go.
func validateServerIP(s string) error {
	if strings.Contains(s, ":") {
		host, _, err := net.SplitHostPort(s)
		if err != nil {
			return fmt.Errorf("invalid --server %q: %w", s, err)
		}
		if net.ParseIP(host) == nil {
			return fmt.Errorf("--server must be an IP address, got hostname %q", host)
		}
		return nil
	}
	if net.ParseIP(s) == nil {
		return fmt.Errorf("--server must be an IP address, got %q", s)
	}
	return nil
}

// buildJobName produces a valid Kubernetes Job name from a hostname.
// Dots are replaced with dashes and the result is truncated to 52 characters
// to leave room for the Kubernetes-appended pod name suffix.
func buildJobName(hostname string) string {
	name := "cure-dns-" + strings.ReplaceAll(hostname, ".", "-")
	if len(name) > 52 {
		name = name[:52]
	}
	// Trim any trailing dashes that may result from truncation.
	name = strings.TrimRight(name, "-")
	return name
}
