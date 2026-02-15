package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mrlm-net/cure/internal/commands"
	"github.com/mrlm-net/cure/pkg/terminal"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantErr    bool
		errContain string
	}{
		{
			name:    "version command",
			args:    []string{"version"},
			wantErr: false,
		},
		{
			name:    "help command",
			args:    []string{"help"},
			wantErr: false,
		},
		{
			name:    "help version",
			args:    []string{"help", "version"},
			wantErr: false,
		},
		{
			name:       "no args",
			args:       nil,
			wantErr:    true,
			errContain: "no command specified",
		},
		{
			name:       "unknown command",
			args:       []string{"nonexistent"},
			wantErr:    true,
			errContain: "unknown command",
		},
		{
			name:    "trace command help",
			args:    []string{"help", "trace"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContain != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("run() error = %v, want error containing %q", err, tt.errContain)
				}
			}
		})
	}
}

func TestRun_VersionOutput(t *testing.T) {
	var stdout bytes.Buffer

	router := terminal.New(terminal.WithStdout(&stdout))
	router.Register(commands.NewVersionCommand())
	router.Register(terminal.NewHelpCommand(router))

	err := router.RunArgs([]string{"version"})
	if err != nil {
		t.Fatalf("router.RunArgs() error = %v", err)
	}

	got := stdout.String()
	want := "cure version dev"
	if !strings.Contains(got, want) {
		t.Errorf("version output = %q, want to contain %q", got, want)
	}
}

func TestRun_HelpOutput(t *testing.T) {
	var stdout bytes.Buffer

	router := terminal.New(terminal.WithStdout(&stdout))
	router.Register(commands.NewVersionCommand())
	router.Register(terminal.NewHelpCommand(router))

	err := router.RunArgs([]string{"help"})
	if err != nil {
		t.Fatalf("router.RunArgs() error = %v", err)
	}

	got := stdout.String()
	want := "Available commands:"
	if !strings.Contains(got, want) {
		t.Errorf("help output = %q, want to contain %q", got, want)
	}
}

func TestRun_HelpVersionOutput(t *testing.T) {
	var stdout bytes.Buffer

	router := terminal.New(terminal.WithStdout(&stdout))
	router.Register(commands.NewVersionCommand())
	router.Register(terminal.NewHelpCommand(router))

	err := router.RunArgs([]string{"help", "version"})
	if err != nil {
		t.Fatalf("router.RunArgs() error = %v", err)
	}

	got := stdout.String()
	want := "Print version information"
	if !strings.Contains(got, want) {
		t.Errorf("help version output = %q, want to contain %q", got, want)
	}
}
