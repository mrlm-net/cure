package commands

import (
	"bytes"
	"context"
	"testing"

	"github.com/mrlm-net/cure/pkg/terminal"
)

// Compile-time interface check.
var _ terminal.Command = (*VersionCommand)(nil)

func TestVersionCommand_Name(t *testing.T) {
	cmd := &VersionCommand{}
	if got := cmd.Name(); got != "version" {
		t.Errorf("Name() = %q, want %q", got, "version")
	}
}

func TestVersionCommand_Description(t *testing.T) {
	cmd := &VersionCommand{}
	if got := cmd.Description(); got == "" {
		t.Error("Description() returned empty string")
	}
}

func TestVersionCommand_Usage(t *testing.T) {
	cmd := &VersionCommand{}
	if got := cmd.Usage(); got == "" {
		t.Error("Usage() returned empty string")
	}
}

func TestVersionCommand_Flags(t *testing.T) {
	cmd := &VersionCommand{}
	if got := cmd.Flags(); got != nil {
		t.Errorf("Flags() = %v, want nil", got)
	}
}

func TestVersionCommand_Run(t *testing.T) {
	tests := []struct {
		name       string
		wantOutput string
		wantErr    bool
	}{
		{
			name:       "prints version",
			wantOutput: "cure version dev\n",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &VersionCommand{}
			var stdout bytes.Buffer

			tc := &terminal.Context{
				Stdout: &stdout,
			}

			err := cmd.Run(context.Background(), tc)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got := stdout.String(); got != tt.wantOutput {
				t.Errorf("Run() output = %q, want %q", got, tt.wantOutput)
			}
		})
	}
}

func BenchmarkVersionCommand_Run(b *testing.B) {
	cmd := &VersionCommand{}
	var stdout bytes.Buffer
	tc := &terminal.Context{
		Stdout: &stdout,
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stdout.Reset()
		_ = cmd.Run(ctx, tc)
	}
}
