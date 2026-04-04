package claudecode

import (
	"os"
	"testing"
)

// lookupEnv wraps os.Getenv for use in tests.
func lookupEnv(key string) string { return os.Getenv(key) }

// assertArg checks that args contains the key-value pair as consecutive elements.
func assertArg(t *testing.T, args []string, key, value string) {
	t.Helper()
	for i := 0; i < len(args)-1; i++ {
		if args[i] == key && args[i+1] == value {
			return
		}
	}
	t.Errorf("expected args to contain %q %q; args: %v", key, value, args)
}

// assertFlag checks that args contains the given flag string.
func assertFlag(t *testing.T, args []string, flag string) {
	t.Helper()
	for _, a := range args {
		if a == flag {
			return
		}
	}
	t.Errorf("expected args to contain flag %q; args: %v", flag, args)
}
