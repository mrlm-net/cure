package env

import "testing"

// BenchmarkDetect_Cached measures the cost of calling Detect() after the
// singleton has been populated. The cached path should be sub-microsecond
// since it only performs a pointer dereference and struct copy.
func BenchmarkDetect_Cached(b *testing.B) {
	// Warm the cache before timing.
	_ = Detect()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Detect()
	}
}

// BenchmarkHasTool_Hit measures HasTool for a tool that exists (no PATH miss).
func BenchmarkHasTool_Hit(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = HasTool("go")
	}
}

// BenchmarkHasTool_Miss measures HasTool for a tool that does not exist,
// exercising the full PATH search and LookPath failure path.
func BenchmarkHasTool_Miss(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = HasTool("nonexistent-tool-xyz-bench")
	}
}

// BenchmarkIsGitRepo measures the cost of IsGitRepo() starting from the
// current working directory (which is inside a git repository for this
// project, so it should find .git quickly).
func BenchmarkIsGitRepo(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsGitRepo()
	}
}
