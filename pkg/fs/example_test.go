package fs_test

import (
	"fmt"
	"os"

	"github.com/mrlm-net/cure/pkg/fs"
)

// ExampleEnsureDir demonstrates creating a directory hierarchy and then
// checking whether a path exists. Both functions return a clear error when
// something unexpected occurs, making them suitable for use in CLI startup
// paths where a missing directory should be created rather than treated as
// a fatal condition.
func ExampleEnsureDir() {
	dir, err := os.MkdirTemp("", "cure-example-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	target := dir + "/sub/deep"

	// EnsureDir creates the directory and all parents.
	if err := fs.EnsureDir(target, 0o700); err != nil {
		fmt.Println("error:", err)
		return
	}

	// Calling EnsureDir again on an existing directory is a no-op.
	if err := fs.EnsureDir(target, 0o700); err != nil {
		fmt.Println("error:", err)
		return
	}

	exists, err := fs.Exists(target)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(exists)
	// Output:
	// true
}
