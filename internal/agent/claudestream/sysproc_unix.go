//go:build darwin || linux

package claudestream

import "syscall"

// ptyProcAttr returns the SysProcAttr that makes the slave PTY (FD 1)
// the controlling terminal of the child process.  This causes Node.js to
// set process.stdout.isTTY = true and stream tokens progressively.
func ptyProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid:  true, // new session — child becomes session leader
		Setctty: true, // set controlling terminal
		Ctty:    1,    // FD 1 (stdout = slave PTY) is the controlling terminal
	}
}
