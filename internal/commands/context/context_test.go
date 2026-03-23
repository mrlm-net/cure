package ctxcmd

import (
	"testing"

	"github.com/mrlm-net/cure/pkg/terminal"
)

func TestNewContextCommand_Name(t *testing.T) {
	st := newMockStore()
	cmd := NewContextCommand(st)
	if cmd.Name() != "context" {
		t.Errorf("Name() = %q, want %q", cmd.Name(), "context")
	}
}

func TestNewContextCommand_NewAndResumeRegistered(t *testing.T) {
	st := newMockStore()
	cmd := NewContextCommand(st)

	router, ok := cmd.(*terminal.Router)
	if !ok {
		t.Skip("NewContextCommand does not return *terminal.Router — cannot inspect registrations")
	}

	if _, found := router.Lookup("new"); !found {
		t.Error("expected 'new' subcommand to be registered")
	}
	if _, found := router.Lookup("resume"); !found {
		t.Error("expected 'resume' subcommand to be registered")
	}
}

func TestNewContextCommand_ListForkDeleteRegistered(t *testing.T) {
	st := newMockStore()
	cmd := NewContextCommand(st)

	router, ok := cmd.(*terminal.Router)
	if !ok {
		t.Skip("NewContextCommand does not return *terminal.Router — cannot inspect registrations")
	}

	for _, name := range []string{"list", "fork", "delete"} {
		if _, found := router.Lookup(name); !found {
			t.Errorf("expected %q to be registered", name)
		}
	}
}
