// Package local implements OS-level desktop notifications via osascript (macOS)
// or notify-send (Linux).
package local

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sync"

	"github.com/mrlm-net/cure/pkg/notify"
)

// Channel implements notify.Channel for OS-level notifications.
type Channel struct {
	warnOnce sync.Once
}

var _ notify.Channel = (*Channel)(nil)

func (c *Channel) Name() string { return "local" }

func (c *Channel) Send(_ context.Context, n notify.Notification) (string, error) {
	title := fmt.Sprintf("cure: %s", n.SessionName)
	body := n.Summary

	switch runtime.GOOS {
	case "darwin":
		return "", c.macosNotify(title, body)
	case "linux":
		return "", c.linuxNotify(title, body)
	default:
		c.warnOnce.Do(func() {})
		return "", nil // silently skip unsupported OS
	}
}

func (c *Channel) Responses() <-chan notify.Response { return nil }

func (c *Channel) macosNotify(title, body string) error {
	script := fmt.Sprintf(`display notification %q with title %q`, body, title)
	return exec.Command("osascript", "-e", script).Run()
}

func (c *Channel) linuxNotify(title, body string) error {
	if _, err := exec.LookPath("notify-send"); err != nil {
		c.warnOnce.Do(func() {})
		return nil
	}
	return exec.Command("notify-send", "--app-name=cure", title, body).Run()
}
