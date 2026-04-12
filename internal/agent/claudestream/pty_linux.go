//go:build linux

package claudestream

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// tiocgptn is the Linux ioctl that returns the PTY slave number.
// Value: _IOR('T', 0x30, unsigned int) = 0x80045430
const tiocgptn = 0x80045430

// tiocsptlck is the Linux ioctl that unlocks the PTY slave.
// Value: _IOW('T', 0x31, int) = 0x40045431
const tiocsptlck = 0x40045431

// openPTY opens a master/slave PTY pair on Linux and returns the master
// file and the slave device path.  The caller is responsible for closing
// the master when done.
func openPTY() (*os.File, string, error) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, "", err
	}

	// Unlock the slave PTY (required on Linux before the slave can be opened).
	var zero int32
	syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), tiocsptlck, uintptr(unsafe.Pointer(&zero))) //nolint:errcheck

	// Retrieve the slave number.
	var ptn uint32
	if _, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		master.Fd(),
		tiocgptn,
		uintptr(unsafe.Pointer(&ptn)),
	); errno != 0 {
		master.Close()
		return nil, "", errno
	}

	return master, fmt.Sprintf("/dev/pts/%d", ptn), nil
}
