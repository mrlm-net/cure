//go:build !darwin && !linux

package claudestream

import "syscall"

// ptyProcAttr returns nil on platforms that do not support PTY-based
// subprocess control.  The adapter falls back to pipe mode on such platforms.
func ptyProcAttr() *syscall.SysProcAttr { return nil }
