package env_test

import (
	"fmt"

	"github.com/mrlm-net/cure/pkg/env"
)

// Example_detect demonstrates retrieving the cached runtime environment.
// The result is computed once on first call and reused on all subsequent
// calls, so it is safe and cheap to call from multiple places in the program.
func Example_detect() {
	e := env.Detect()
	// OS and Arch are always non-empty.
	fmt.Println(e.OS != "")
	fmt.Println(e.Arch != "")
	// Output:
	// true
	// true
}

// ExampleHasTool demonstrates checking whether an external program is
// available on PATH. Go is always available in test environments.
func ExampleHasTool() {
	fmt.Println(env.HasTool("go"))
	fmt.Println(env.HasTool("this-tool-does-not-exist-xyz-abc"))
	// Output:
	// true
	// false
}
