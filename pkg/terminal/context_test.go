package terminal

import (
	"bytes"
	"flag"
	"io"
	"testing"
)

func TestContext(t *testing.T) {
	tests := []struct {
		name   string
		ctx    Context
		check  func(t *testing.T, c *Context)
	}{
		{
			name: "all fields populated",
			ctx: Context{
				Args:   []string{"arg1", "arg2"},
				Flags:  flag.NewFlagSet("test", flag.ContinueOnError),
				Stdout: &bytes.Buffer{},
				Stderr: &bytes.Buffer{},
			},
			check: func(t *testing.T, c *Context) {
				t.Helper()
				if len(c.Args) != 2 {
					t.Errorf("Args length = %d, want 2", len(c.Args))
				}
				if c.Args[0] != "arg1" || c.Args[1] != "arg2" {
					t.Errorf("Args = %v, want [arg1 arg2]", c.Args)
				}
				if c.Flags == nil {
					t.Error("Flags is nil, want non-nil FlagSet")
				}
				if c.Stdout == nil {
					t.Error("Stdout is nil, want non-nil Writer")
				}
				if c.Stderr == nil {
					t.Error("Stderr is nil, want non-nil Writer")
				}
			},
		},
		{
			name: "empty args",
			ctx: Context{
				Args:   []string{},
				Flags:  flag.NewFlagSet("test", flag.ContinueOnError),
				Stdout: &bytes.Buffer{},
				Stderr: &bytes.Buffer{},
			},
			check: func(t *testing.T, c *Context) {
				t.Helper()
				if len(c.Args) != 0 {
					t.Errorf("Args length = %d, want 0", len(c.Args))
				}
			},
		},
		{
			name: "nil flags",
			ctx: Context{
				Args:   []string{"arg1"},
				Flags:  nil,
				Stdout: &bytes.Buffer{},
				Stderr: &bytes.Buffer{},
			},
			check: func(t *testing.T, c *Context) {
				t.Helper()
				if c.Flags != nil {
					t.Error("Flags is non-nil, want nil for command with no flags")
				}
			},
		},
		{
			name: "discard writers",
			ctx: Context{
				Args:   nil,
				Flags:  nil,
				Stdout: io.Discard,
				Stderr: io.Discard,
			},
			check: func(t *testing.T, c *Context) {
				t.Helper()
				n, err := c.Stdout.Write([]byte("silent"))
				if err != nil {
					t.Errorf("Stdout.Write error = %v", err)
				}
				if n != 6 {
					t.Errorf("Stdout.Write n = %d, want 6", n)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, &tt.ctx)
		})
	}
}

func TestContext_StdoutStderrIndependence(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	c := &Context{
		Stdout: stdout,
		Stderr: stderr,
	}

	if _, err := c.Stdout.Write([]byte("out")); err != nil {
		t.Fatalf("Stdout.Write error = %v", err)
	}
	if _, err := c.Stderr.Write([]byte("err")); err != nil {
		t.Fatalf("Stderr.Write error = %v", err)
	}

	if got := stdout.String(); got != "out" {
		t.Errorf("Stdout = %q, want %q", got, "out")
	}
	if got := stderr.String(); got != "err" {
		t.Errorf("Stderr = %q, want %q", got, "err")
	}
}

func TestContext_WriterCapture(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	c := &Context{
		Args:   []string{"file.yaml"},
		Flags:  flag.NewFlagSet("generate", flag.ContinueOnError),
		Stdout: stdout,
		Stderr: stderr,
	}

	// Simulate command writing output
	if _, err := c.Stdout.Write([]byte("generated: file.yaml\n")); err != nil {
		t.Fatalf("Stdout.Write error = %v", err)
	}
	if _, err := c.Stderr.Write([]byte("warning: overwriting existing file\n")); err != nil {
		t.Fatalf("Stderr.Write error = %v", err)
	}

	wantOut := "generated: file.yaml\n"
	if got := stdout.String(); got != wantOut {
		t.Errorf("Stdout = %q, want %q", got, wantOut)
	}

	wantErr := "warning: overwriting existing file\n"
	if got := stderr.String(); got != wantErr {
		t.Errorf("Stderr = %q, want %q", got, wantErr)
	}
}
