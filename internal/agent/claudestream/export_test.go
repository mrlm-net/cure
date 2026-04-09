package claudestream

import (
	"os"
	"testing"
)

// lookupEnv wraps os.Getenv for use in tests.
func lookupEnv(key string) string { return os.Getenv(key) }

// assertArg checks that args contains key followed immediately by value.
func assertArg(t *testing.T, args []string, key, value string) {
	t.Helper()
	for i := 0; i < len(args)-1; i++ {
		if args[i] == key && args[i+1] == value {
			return
		}
	}
	t.Errorf("expected args to contain %q %q; args: %v", key, value, args)
}

// assertFlag checks that args contains the given flag.
func assertFlag(t *testing.T, args []string, flag string) {
	t.Helper()
	for _, a := range args {
		if a == flag {
			return
		}
	}
	t.Errorf("expected args to contain flag %q; args: %v", flag, args)
}

// assertNoFlag checks that args does NOT contain the given flag.
func assertNoFlag(t *testing.T, args []string, flag string) {
	t.Helper()
	for _, a := range args {
		if a == flag {
			t.Errorf("expected args NOT to contain flag %q; args: %v", flag, args)
			return
		}
	}
}
