package generate

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mrlm-net/cure/pkg/terminal"
)

func newK8sJobContext(stdout *bytes.Buffer) *terminal.Context {
	return &terminal.Context{Stdout: stdout}
}

func TestK8sJobCommand_Run_MissingCureCommand(t *testing.T) {
	cmd := &K8sJobCommand{}
	var buf bytes.Buffer
	err := cmd.Run(context.Background(), newK8sJobContext(&buf))
	if err == nil {
		t.Fatal("expected error for missing --cure-command, got nil")
	}
	if !strings.Contains(err.Error(), "--cure-command") {
		t.Errorf("error %q does not mention --cure-command", err.Error())
	}
}

func TestK8sJobCommand_Run_DefaultOutput(t *testing.T) {
	cmd := &K8sJobCommand{
		cureCommand: "trace dns myservice.blob.core.windows.net --count 30",
		namespace:   "default",
		image:       "ghcr.io/mrlm-net/cure:latest",
	}
	var buf bytes.Buffer
	if err := cmd.Run(context.Background(), newK8sJobContext(&buf)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"kind: Job",
		"namespace: default",
		"ghcr.io/mrlm-net/cure:latest",
		`- "trace"`,
		`- "dns"`,
		`- "myservice.blob.core.windows.net"`,
		"job-name: cure-trace-dns",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestK8sJobCommand_Run_HTTPTrace(t *testing.T) {
	cmd := &K8sJobCommand{
		cureCommand: "trace http https://api.internal.example.com",
		namespace:   "monitoring",
		image:       "ghcr.io/mrlm-net/cure:latest",
	}
	var buf bytes.Buffer
	if err := cmd.Run(context.Background(), newK8sJobContext(&buf)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `- "trace"`) || !strings.Contains(out, `- "http"`) {
		t.Error("output missing trace http args")
	}
	if !strings.Contains(out, "namespace: monitoring") {
		t.Error("output missing namespace")
	}
}

func TestK8sJobCommand_Run_WithNodeSelector(t *testing.T) {
	cmd := &K8sJobCommand{
		cureCommand:  "trace dns api.example.com",
		namespace:    "default",
		image:        "ghcr.io/mrlm-net/cure:latest",
		nodeSelector: "agentpool=openaisvc",
	}
	var buf bytes.Buffer
	if err := cmd.Run(context.Background(), newK8sJobContext(&buf)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "nodeSelector:") {
		t.Error("output missing nodeSelector")
	}
	if !strings.Contains(out, "agentpool: openaisvc") {
		t.Error("output missing agentpool selector")
	}
}

func TestK8sJobCommand_Run_WithToleration(t *testing.T) {
	cmd := &K8sJobCommand{
		cureCommand: "trace dns api.example.com",
		namespace:   "default",
		image:       "ghcr.io/mrlm-net/cure:latest",
		toleration:  "kubernetes.azure.com/scalesetpriority=spot:NoSchedule",
	}
	var buf bytes.Buffer
	if err := cmd.Run(context.Background(), newK8sJobContext(&buf)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "tolerations:") {
		t.Error("output missing tolerations")
	}
	if !strings.Contains(out, `key: "kubernetes.azure.com/scalesetpriority"`) {
		t.Error("output missing toleration key")
	}
	if !strings.Contains(out, "operator: Equal") {
		t.Error("output missing Equal operator")
	}
	if !strings.Contains(out, `value: "spot"`) {
		t.Error("output missing toleration value")
	}
}

func TestK8sJobCommand_Run_JobNameOverride(t *testing.T) {
	cmd := &K8sJobCommand{
		cureCommand: "trace dns api.example.com",
		jobName:     "my-custom-job",
		namespace:   "default",
		image:       "ghcr.io/mrlm-net/cure:latest",
	}
	var buf bytes.Buffer
	if err := cmd.Run(context.Background(), newK8sJobContext(&buf)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "name: my-custom-job") {
		t.Error("output missing overridden job name")
	}
}

func TestK8sJobCommand_Run_InvalidNodeSelector(t *testing.T) {
	cmd := &K8sJobCommand{
		cureCommand:  "trace dns api.example.com",
		nodeSelector: "notakeyvalue",
	}
	var buf bytes.Buffer
	err := cmd.Run(context.Background(), newK8sJobContext(&buf))
	if err == nil {
		t.Fatal("expected error for invalid node selector, got nil")
	}
}

func TestK8sJobCommand_Run_InvalidToleration(t *testing.T) {
	cmd := &K8sJobCommand{
		cureCommand: "trace dns api.example.com",
		toleration:  "noeffect",
	}
	var buf bytes.Buffer
	err := cmd.Run(context.Background(), newK8sJobContext(&buf))
	if err == nil {
		t.Fatal("expected error for invalid toleration, got nil")
	}
}

func TestK8sJobCommand_Run_CustomImage(t *testing.T) {
	cmd := &K8sJobCommand{
		cureCommand: "trace dns api.example.com",
		image:       "myacr.azurecr.io/cure:v0.5.0",
	}
	var buf bytes.Buffer
	if err := cmd.Run(context.Background(), newK8sJobContext(&buf)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(buf.String(), "myacr.azurecr.io/cure:v0.5.0") {
		t.Error("output missing custom image")
	}
}

func TestDeriveJobName(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    string
	}{
		{"dns trace", "trace dns myservice.blob.core.windows.net --count 30", "cure-trace-dns"},
		{"http trace", "trace http https://api.example.com", "cure-trace-http"},
		{"tcp trace", "trace tcp 10.0.0.5:6379", "cure-trace-tcp"},
		{"single word", "version", "cure-version"},
		{"uppercase sanitized", "TRACE DNS example.com", "cure-trace-dns"},
		{"flags stop token collection", "trace --dry-run dns", "cure-trace"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveJobName(tt.command)
			if got != tt.want {
				t.Errorf("deriveJobName(%q) = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
}
