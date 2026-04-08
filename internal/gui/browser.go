package gui

import (
	"os/exec"
	"runtime"
)

// OpenBrowser opens the given URL in the system's default browser.
// Returns an error if the browser command fails to start. On unsupported
// operating systems it returns nil silently — the URL is already printed
// to stdout by the caller.
func OpenBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", url).Start()
	default:
		return nil
	}
}
