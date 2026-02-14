package main

import "testing"

func TestRun_Version(t *testing.T) {
	err := run([]string{"version"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRun_Help(t *testing.T) {
	err := run([]string{"help"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRun_NoArgs(t *testing.T) {
	err := run(nil)
	if err == nil {
		t.Fatal("expected error for empty args")
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	err := run([]string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}
