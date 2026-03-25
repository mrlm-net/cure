package fs_test

import (
	"os"
	"path/filepath"
	"testing"

	cfs "github.com/mrlm-net/cure/pkg/fs"
)

// payload1KB is representative content for a 1 KB write (the benchmark target).
var payload1KB = func() []byte {
	b := make([]byte, 1024)
	for i := range b {
		b[i] = byte(i % 256)
	}
	return b
}()

// BenchmarkAtomicWrite_1KB measures the overhead of AtomicWrite for a 1 KB
// payload versus the plain os.WriteFile baseline. The target is <10 ms of
// overhead per operation.
func BenchmarkAtomicWrite_1KB(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "bench.bin")

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		if err := cfs.AtomicWrite(path, payload1KB, 0o644); err != nil {
			b.Fatalf("AtomicWrite: %v", err)
		}
	}
}

// BenchmarkOSWriteFile_1KB is the baseline to compare AtomicWrite against.
func BenchmarkOSWriteFile_1KB(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "bench.bin")

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		if err := os.WriteFile(path, payload1KB, 0o644); err != nil {
			b.Fatalf("WriteFile: %v", err)
		}
	}
}

// BenchmarkAtomicWrite_64KB exercises a larger payload to show how overhead
// scales with content size.
func BenchmarkAtomicWrite_64KB(b *testing.B) {
	payload := make([]byte, 64*1024)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	dir := b.TempDir()
	path := filepath.Join(dir, "bench64.bin")

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		if err := cfs.AtomicWrite(path, payload, 0o644); err != nil {
			b.Fatalf("AtomicWrite: %v", err)
		}
	}
}

// BenchmarkOSWriteFile_64KB is the baseline for the larger payload.
func BenchmarkOSWriteFile_64KB(b *testing.B) {
	payload := make([]byte, 64*1024)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	dir := b.TempDir()
	path := filepath.Join(dir, "bench64.bin")

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		if err := os.WriteFile(path, payload, 0o644); err != nil {
			b.Fatalf("WriteFile: %v", err)
		}
	}
}
