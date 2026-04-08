package gui

import (
	"runtime"
	"testing"
)

func TestOpenBrowser(t *testing.T) {
	t.Run("does not panic", func(t *testing.T) {
		// We cannot actually open a browser in CI/tests, but we verify
		// the function doesn't panic regardless of OS.
		// On darwin this will start "open" which returns quickly;
		// on other OS it will either start the command or return nil.
		_ = OpenBrowser("http://127.0.0.1:12345")
	})

	t.Run("returns nil on unsupported OS", func(t *testing.T) {
		// On any supported OS (darwin, linux, windows) the function
		// attempts exec.Command.Start which may succeed or fail.
		// We just verify the function is callable without panic.
		if runtime.GOOS != "darwin" && runtime.GOOS != "linux" && runtime.GOOS != "windows" {
			err := OpenBrowser("http://127.0.0.1:12345")
			if err != nil {
				t.Errorf("expected nil on unsupported OS, got: %v", err)
			}
		}
	})
}
