//go:build darwin

package claudestream

import (
	"os"
	"strings"
	"syscall"
	"unsafe"
)

// tiocptyname is the macOS ioctl that returns the slave PTY device path.
// Value: _IOC(IOC_OUT, 't', 83, 128) = 0x40807453
const tiocptyname = 0x40807453

// openPTY opens a master/slave PTY pair on macOS and returns the master
// file and the slave device path.  The caller is responsible for closing
// the master when done.
func openPTY() (*os.File, string, error) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, "", err
	}

	// Retrieve the slave device path via TIOCPTYNAME.
	var name [128]byte
	if _, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		master.Fd(),
		tiocptyname,
		uintptr(unsafe.Pointer(&name[0])),
	); errno != 0 {
		master.Close()
		return nil, "", errno
	}

	slaveName := strings.TrimRight(string(name[:]), "\x00")
	return master, slaveName, nil
}
