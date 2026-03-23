package agent_test

import (
	"strings"
	"testing"

	"github.com/mrlm-net/cure/pkg/agent"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty string", "", 0},
		{"four chars", "abcd", 1},
		{"eight chars", "abcdefgh", 2},
		{"three chars truncates", "abc", 0},
		{"twelve chars", "abcdefghijkl", 3},
		{"typical sentence", "The quick brown fox", 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := agent.EstimateTokens(tt.input)
			if got != tt.want {
				t.Errorf("EstimateTokens(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func BenchmarkEstimateTokens(b *testing.B) {
	const base = "The quick brown fox jumps over the lazy dog. "
	cases := []struct {
		name string
		n    int
	}{
		{"1msg", 1},
		{"10msg", 10},
		{"100msg", 100},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			content := strings.Repeat(base, tc.n)
			b.ResetTimer()
			for range b.N {
				agent.EstimateTokens(content)
			}
		})
	}
}
