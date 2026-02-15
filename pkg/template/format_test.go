package template

import "testing"

func TestFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "trailing whitespace removed",
			input: "line1   \nline2\t\n",
			want:  "line1\nline2\n",
		},
		{
			name:  "windows line endings normalized",
			input: "line1\r\nline2\r\n",
			want:  "line1\nline2\n",
		},
		{
			name:  "three blank lines reduced to two",
			input: "line1\n\n\n\nline2\n",
			want:  "line1\n\nline2\n",
		},
		{
			name:  "four blank lines reduced to two",
			input: "line1\n\n\n\n\nline2\n",
			want:  "line1\n\nline2\n",
		},
		{
			name:  "two blank lines preserved",
			input: "line1\n\n\nline2\n",
			want:  "line1\n\nline2\n",
		},
		{
			name:  "one blank line preserved",
			input: "line1\n\nline2\n",
			want:  "line1\n\nline2\n",
		},
		{
			name:  "ensures final newline",
			input: "line1\nline2",
			want:  "line1\nline2\n",
		},
		{
			name:  "empty string gets newline",
			input: "",
			want:  "\n",
		},
		{
			name:  "mixed trailing whitespace",
			input: "line1 \t \nline2  \n",
			want:  "line1\nline2\n",
		},
		{
			name:  "preserves indent",
			input: "  line1\n    line2\n",
			want:  "  line1\n    line2\n",
		},
		{
			name:  "complex case with all issues",
			input: "line1  \r\n\n\n\n  line2\t\r\nline3",
			want:  "line1\n\n  line2\nline3\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Format(tt.input)
			if got != tt.want {
				t.Errorf("Format() = %q, want %q", got, tt.want)
			}
		})
	}
}
