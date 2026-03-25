package style_test

import (
	"fmt"

	"github.com/mrlm-net/cure/pkg/style"
)

// ExampleReset demonstrates stripping ANSI escape codes from a styled string.
// This is useful when computing display widths or writing to log files where
// escape codes would appear as literal characters.
func ExampleReset() {
	styled := style.Bold(style.Red("FAIL"))
	plain := style.Reset(styled)
	fmt.Println(plain)
	// Output:
	// FAIL
}
