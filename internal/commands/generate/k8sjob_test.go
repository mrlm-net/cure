package generate

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mrlm-net/cure/pkg/config"
	"github.com/mrlm-net/cure/pkg/terminal"
)

func newK8sJobContext() (*bytes.Buffer, *bytes.Buffer, *terminal.Context) {
	var stdout, stderr bytes.Buffer
	tc := &terminal.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Config: config.NewConfig(),
	}
	return &stdout, &stderr, tc
}

func TestK8sJobCommand_Run_MissingHostname(t *testing.T) {
	cmd := &K8sJobCommand{
		namespace: "default",
		image:     "golang:1.25-alpine",
		version:   "latest",
		count:     30,
		interval:  10,
		timeout:   30,
	}

	_, _, tc := newK8sJobContext()
	err := cmd.Run(context.Background(), tc)
	if err == nil {
		t.Fatal("expected error for missing --hostname, got nil")
	}
	if !strings.Contains(err.Error(), "--hostname is required") {
		t.Errorf("expected '--hostname is required' in error, got: %v", err)
	}
}

func TestK8sJobCommand_Run_InvalidServer(t *testing.T) {
	tests := []struct {
		name   string
		server string
	}{
		{"plain hostname", "myserver.example.com"},
		{"hostname with port", "myserver.example.com:53"},
		{"bare word", "dns"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &K8sJobCommand{
				hostname:  "api.example.com",
				namespace: "default",
				image:     "golang:1.25-alpine",
				version:   "latest",
				count:     30,
				interval:  10,
				timeout:   30,
				server:    tt.server,
			}

			_, _, tc := newK8sJobContext()
			err := cmd.Run(context.Background(), tc)
			if err == nil {
				t.Fatalf("expected error for server=%q, got nil", tt.server)
			}
			if !strings.Contains(err.Error(), "--server must be an IP address") {
				t.Errorf("expected '--server must be an IP address' in error, got: %v", err)
			}
		})
	}
}

func TestK8sJobCommand_Run_DefaultOutput(t *testing.T) {
	cmd := &K8sJobCommand{
		hostname:  "api.example.com",
		namespace: "default",
		image:     "golang:1.25-alpine",
		version:   "latest",
		count:     30,
		interval:  10,
		timeout:   30,
	}

	stdout, _, tc := newK8sJobContext()
	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	got := stdout.String()
	checks := []string{
		"kind: Job",
		"cure-dns-api-example-com",
		"namespace: default",
		"api.example.com",
		"golang:1.25-alpine",
		"restartPolicy: Never",
		"backoffLimit: 0",
		"ttlSecondsAfterFinished: 600",
		"--count=30",
		"--interval=10",
		"--timeout=30",
	}
	for _, want := range checks {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, got)
		}
	}

	// Server flag must NOT appear when server is empty.
	if strings.Contains(got, "--server=") {
		t.Errorf("output should not contain --server= when server is empty\ngot:\n%s", got)
	}
}

func TestK8sJobCommand_Run_WithServer(t *testing.T) {
	cmd := &K8sJobCommand{
		hostname:  "myservice.default.svc.cluster.local",
		namespace: "default",
		image:     "golang:1.25-alpine",
		version:   "latest",
		count:     10,
		interval:  5,
		timeout:   15,
		server:    "168.63.129.16",
	}

	stdout, _, tc := newK8sJobContext()
	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "--server=168.63.129.16") {
		t.Errorf("output missing --server=168.63.129.16\ngot:\n%s", got)
	}
}

func TestK8sJobCommand_Run_CustomNamespace(t *testing.T) {
	cmd := &K8sJobCommand{
		hostname:  "api.example.com",
		namespace: "monitoring",
		image:     "golang:1.25-alpine",
		version:   "latest",
		count:     30,
		interval:  10,
		timeout:   30,
	}

	stdout, _, tc := newK8sJobContext()
	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "namespace: monitoring") {
		t.Errorf("output missing 'namespace: monitoring'\ngot:\n%s", got)
	}
}

func TestK8sJobCommand_Run_JobNameSanitization(t *testing.T) {
	tests := []struct {
		name        string
		hostname    string
		wantJobName string
	}{
		{
			name:        "simple hostname",
			hostname:    "api.example.com",
			wantJobName: "cure-dns-api-example-com",
		},
		{
			name:        "subdomain",
			hostname:    "myservice.default.svc.cluster.local",
			wantJobName: "cure-dns-myservice-default-svc-cluster-local",
		},
		{
			name:        "no dots",
			hostname:    "localhost",
			wantJobName: "cure-dns-localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildJobName(tt.hostname)
			if got != tt.wantJobName {
				t.Errorf("buildJobName(%q) = %q, want %q", tt.hostname, got, tt.wantJobName)
			}
		})
	}
}

func TestK8sJobCommand_Run_JobNameTruncation(t *testing.T) {
	// A very long hostname should be truncated to ≤52 chars.
	longHostname := "very-long-service-name.my-very-long-namespace.svc.cluster.local"
	got := buildJobName(longHostname)
	if len(got) > 52 {
		t.Errorf("buildJobName() result too long: %d chars, want ≤52\ngot: %s", len(got), got)
	}
	if strings.HasSuffix(got, "-") {
		t.Errorf("buildJobName() result must not end with a dash, got: %s", got)
	}
}

func TestK8sJobCommand_Run_OutputFile(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "job.yaml")

	cmd := &K8sJobCommand{
		hostname:  "api.example.com",
		namespace: "default",
		image:     "golang:1.25-alpine",
		version:   "v0.5.0",
		count:     5,
		interval:  2,
		timeout:   10,
		output:    outPath,
	}

	stdout, _, tc := newK8sJobContext()
	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	// Stdout should confirm the file was written.
	if !strings.Contains(stdout.String(), outPath) {
		t.Errorf("stdout should mention output path, got: %s", stdout.String())
	}

	// File should exist and contain expected content.
	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	got := string(content)
	checks := []string{
		"kind: Job",
		"cure-dns-api-example-com",
		"v0.5.0",
		"--count=5",
	}
	for _, want := range checks {
		if !strings.Contains(got, want) {
			t.Errorf("file missing %q\ngot:\n%s", want, got)
		}
	}
}

func TestK8sJobCommand_FlagsDefaults(t *testing.T) {
	cmd := &K8sJobCommand{}
	fs := cmd.Flags()
	// Parse empty args to get defaults.
	if err := fs.Parse([]string{}); err != nil {
		t.Fatalf("Flags().Parse() error: %v", err)
	}

	if cmd.namespace != "default" {
		t.Errorf("default namespace = %q, want %q", cmd.namespace, "default")
	}
	if cmd.image != "golang:1.25-alpine" {
		t.Errorf("default image = %q, want %q", cmd.image, "golang:1.25-alpine")
	}
	if cmd.version != "latest" {
		t.Errorf("default version = %q, want %q", cmd.version, "latest")
	}
	if cmd.count != 30 {
		t.Errorf("default count = %d, want 30", cmd.count)
	}
	if cmd.interval != 10 {
		t.Errorf("default interval = %d, want 10", cmd.interval)
	}
	if cmd.timeout != 30 {
		t.Errorf("default timeout = %d, want 30", cmd.timeout)
	}
}

func TestK8sJobCommand_ValidServerIP(t *testing.T) {
	tests := []struct {
		name    string
		server  string
		wantErr bool
	}{
		{"plain IPv4", "168.63.129.16", false},
		{"IPv4 with port", "168.63.129.16:53", false},
		{"IPv6 with brackets and port", "[::1]:53", false},
		{"hostname rejected", "dns.google", true},
		{"hostname with port rejected", "dns.google:53", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServerIP(tt.server)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateServerIP(%q) error = %v, wantErr %v", tt.server, err, tt.wantErr)
			}
		})
	}
}
