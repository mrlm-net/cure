//go:build linux || darwin

package ctxcmd

import "syscall"

// isatty reports whether fd refers to a character device (interactive terminal).
// It uses syscall.Fstat and checks for S_IFCHR — the character device type bit.
func isatty(fd uintptr) bool {
	var stat syscall.Stat_t
	if err := syscall.Fstat(int(fd), &stat); err != nil {
		return false
	}
	return stat.Mode&syscall.S_IFMT == syscall.S_IFCHR
}
