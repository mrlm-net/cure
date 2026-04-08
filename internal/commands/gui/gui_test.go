package guicmd

import (
	"testing"

	"github.com/mrlm-net/cure/pkg/config"
	"github.com/mrlm-net/cure/pkg/doctor"
)

func TestGUICommandName(t *testing.T) {
	cmd := NewGUICommand(nil, nil, nil)
	if got := cmd.Name(); got != "gui" {
		t.Errorf("Name() = %q, want %q", got, "gui")
	}
}

func TestGUICommandDescription(t *testing.T) {
	cmd := NewGUICommand(nil, nil, nil)
	if got := cmd.Description(); got == "" {
		t.Error("Description() is empty, want non-empty string")
	}
}

func TestGUICommandUsage(t *testing.T) {
	cmd := NewGUICommand(nil, nil, nil)
	if got := cmd.Usage(); got == "" {
		t.Error("Usage() is empty, want non-empty string")
	}
}

func TestGUICommandFlags(t *testing.T) {
	cmd := NewGUICommand(nil, nil, nil)
	fs := cmd.Flags()
	if fs == nil {
		t.Fatal("Flags() returned nil")
	}

	t.Run("port flag exists", func(t *testing.T) {
		f := fs.Lookup("port")
		if f == nil {
			t.Fatal("flag 'port' not found")
		}
		if f.DefValue != "0" {
			t.Errorf("port default = %q, want %q", f.DefValue, "0")
		}
	})

	t.Run("no-browser flag exists", func(t *testing.T) {
		f := fs.Lookup("no-browser")
		if f == nil {
			t.Fatal("flag 'no-browser' not found")
		}
		if f.DefValue != "false" {
			t.Errorf("no-browser default = %q, want %q", f.DefValue, "false")
		}
	})
}

func TestGUICommandConstructor(t *testing.T) {
	cfgData := config.ConfigObject{"key": "value"}
	checks := []doctor.CheckFunc{
		func() doctor.CheckResult {
			return doctor.CheckResult{Name: "test", Status: doctor.CheckPass}
		},
	}

	cmd := NewGUICommand(cfgData, checks, nil)
	gc, ok := cmd.(*GUICommand)
	if !ok {
		t.Fatalf("NewGUICommand returned %T, want *GUICommand", cmd)
	}
	if gc.cfgData == nil {
		t.Error("cfgData is nil after construction")
	}
	if len(gc.checks) != 1 {
		t.Errorf("checks length = %d, want 1", len(gc.checks))
	}
	if gc.store != nil {
		t.Error("store should be nil when nil was passed")
	}
}
