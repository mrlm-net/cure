package generate

import (
	"context"
	"flag"
	"fmt"
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

// K8sJobCommand generates a Kubernetes Job manifest that runs any cure command in-cluster.
type K8sJobCommand struct {
	cureCommand  string // full cure subcommand + args, e.g. "trace dns example.com --count 30"
	jobName      string // override auto-derived job name
	namespace    string
	image        string
	version      string
	nodeSelector string // comma-separated key=value pairs
	toleration   string // comma-separated key=value:effect or key:effect specs
	output       string // "" = stdout
}

func (c *K8sJobCommand) Name() string { return "k8s-job" }
func (c *K8sJobCommand) Description() string {
	return "Generate a Kubernetes Job manifest to run any cure command in-cluster"
}
func (c *K8sJobCommand) Usage() string {
	return `Usage: cure generate k8s-job [flags]

Generate a Kubernetes Job manifest that runs any cure command inside a cluster.
Useful for diagnosing in-cluster issues (DNS flapping, HTTP connectivity, TCP
reachability) from within the pod network of a specific namespace or node pool.

Required flags:
  --cure-command   Full cure subcommand and args (required)

Optional flags:
  --job-name       Override auto-derived job name
  --namespace      Kubernetes namespace (default: default)
  --image          Container image (default: golang:1.25-alpine)
  --cure-version   cure version to install, e.g. v0.5.0 (default: latest)
  --node-selector  Comma-separated key=value node labels (e.g. agentpool=gpupool)
  --toleration     Comma-separated toleration specs: key=value:effect or key:effect
  --output         Output file path; empty = stdout (default: "")

Examples:
  # DNS trace — print manifest to stdout
  cure generate k8s-job \
    --cure-command "trace dns myservice.blob.core.windows.net --count 60 --interval 10 --server 168.63.129.16" \
    --namespace openai-svc

  # HTTP trace — target a specific AKS node pool
  cure generate k8s-job \
    --cure-command "trace http https://api.internal.example.com" \
    --namespace monitoring \
    --node-selector "agentpool=appnodes"

  # Spot node pool (with taint toleration)
  cure generate k8s-job \
    --cure-command "trace dns myservice.blob.core.windows.net --count 30" \
    --node-selector "kubernetes.azure.com/agentpool=spotnodes" \
    --toleration "kubernetes.azure.com/scalesetpriority=spot:NoSchedule"

  # Apply directly via kubectl
  cure generate k8s-job \
    --cure-command "trace dns api.example.com" \
    --namespace default | kubectl apply -f -
`
}

func (c *K8sJobCommand) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("k8s-job", flag.ContinueOnError)
	fs.StringVar(&c.cureCommand, "cure-command", "", "Full cure subcommand and args (required)")
	fs.StringVar(&c.jobName, "job-name", "", "Override auto-derived job name")
	fs.StringVar(&c.namespace, "namespace", "default", "Target Kubernetes namespace")
	fs.StringVar(&c.image, "image", "golang:1.25-alpine", "Container image")
	fs.StringVar(&c.version, "cure-version", "latest", "cure version to install (e.g. v0.5.0 or latest)")
	fs.StringVar(&c.nodeSelector, "node-selector", "", "Comma-separated key=value node labels (e.g. agentpool=gpupool)")
	fs.StringVar(&c.toleration, "toleration", "", "Comma-separated toleration specs: key=value:effect or key:effect")
	fs.StringVar(&c.output, "output", "", "Output file path (empty = stdout)")
	return fs
}

func (c *K8sJobCommand) Run(ctx context.Context, tc *terminal.Context) error {
	if c.cureCommand == "" {
		return fmt.Errorf("--cure-command is required")
	}

	jobName := c.jobName
	if jobName == "" {
		jobName = deriveJobName(c.cureCommand)
	}

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
		"CureCommand":  c.cureCommand,
		"Image":        c.image,
		"Version":      c.version,
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

// deriveJobName builds a Kubernetes-safe job name from the cure command string.
// Collects only pure-alpha tokens (subcommand words like "trace", "dns", "http"),
// stopping at the first argument that contains non-letter characters (hostnames,
// URLs, IPs, ports) or at a flag prefix "-".
// Example: "trace dns example.com --count 30" → "cure-trace-dns"
func deriveJobName(cureCommand string) string {
	parts := strings.Fields(cureCommand)
	var tokens []string
	for _, p := range parts {
		if strings.HasPrefix(p, "-") {
			break
		}
		// Stop at non-alpha tokens (hostnames, URLs, IPs, etc.)
		if !isAlpha(p) {
			break
		}
		tokens = append(tokens, p)
		if len(tokens) == 3 {
			break
		}
	}
	name := "cure-" + strings.Join(tokens, "-")
	// Sanitize: replace dots, slashes, and other non-DNS characters with dashes.
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + 32) // toLower
		default:
			b.WriteRune('-')
		}
	}
	result := strings.Trim(b.String(), "-")
	// Kubernetes name max length is 63 characters.
	if len(result) > 63 {
		result = strings.TrimRight(result[:63], "-")
	}
	return result
}

// isAlpha reports whether s contains only ASCII letters.
func isAlpha(s string) bool {
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
			return false
		}
	}
	return len(s) > 0
}

// parseNodeSelector parses a comma-separated "key=value" string into a map.
// Returns nil when s is empty so the template can test {{if .NodeSelector}}.
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
		if k == "" || v == "" {
			return k8sToleration{}, fmt.Errorf("invalid --toleration %q: key and value must be non-empty", spec)
		}
		return k8sToleration{Key: k, Value: v, Operator: "Equal", Effect: effect}, nil
	}
	if keyPart == "" {
		return k8sToleration{}, fmt.Errorf("invalid --toleration %q: key is empty", spec)
	}
	return k8sToleration{Key: keyPart, Operator: "Exists", Effect: effect}, nil
}

