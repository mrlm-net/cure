package generate

import (
	"strings"
	"testing"
)

func TestPrompterRequired(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		defaultVal string
		want       string
		wantErr    bool
	}{
		{
			name:       "user provides value",
			input:      "myapp\n",
			defaultVal: "",
			want:       "myapp",
			wantErr:    false,
		},
		{
			name:       "user accepts default",
			input:      "\n",
			defaultVal: "default",
			want:       "default",
			wantErr:    false,
		},
		{
			name:       "user provides value after empty attempt",
			input:      "\nmyapp\n",
			defaultVal: "",
			want:       "myapp",
			wantErr:    false,
		},
		{
			name:       "whitespace is trimmed",
			input:      "  myapp  \n",
			defaultVal: "",
			want:       "myapp",
			wantErr:    false,
		},
		{
			name:       "default is trimmed",
			input:      "\n",
			defaultVal: "  default  ",
			want:       "  default  ",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout strings.Builder
			stdin := strings.NewReader(tt.input)
			p := NewPrompter(&stdout, stdin)

			got, err := p.Required("Enter name:", tt.defaultVal)
			if (err != nil) != tt.wantErr {
				t.Errorf("Required() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Required() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrompterOptional(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		defaultVal string
		want       string
	}{
		{
			name:       "user provides value",
			input:      "myvalue\n",
			defaultVal: "default",
			want:       "myvalue",
		},
		{
			name:       "user accepts default",
			input:      "\n",
			defaultVal: "default",
			want:       "default",
		},
		{
			name:       "whitespace is trimmed",
			input:      "  myvalue  \n",
			defaultVal: "default",
			want:       "myvalue",
		},
		{
			name:       "empty input returns default",
			input:      "   \n",
			defaultVal: "default",
			want:       "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout strings.Builder
			stdin := strings.NewReader(tt.input)
			p := NewPrompter(&stdout, stdin)

			got, err := p.Optional("Enter value:", tt.defaultVal)
			if err != nil {
				t.Errorf("Optional() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Optional() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrompterConfirm(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    bool
		wantErr bool
	}{
		{
			name:    "y confirms",
			input:   "y\n",
			want:    true,
			wantErr: false,
		},
		{
			name:    "yes confirms",
			input:   "yes\n",
			want:    true,
			wantErr: false,
		},
		{
			name:    "Y confirms (case insensitive)",
			input:   "Y\n",
			want:    true,
			wantErr: false,
		},
		{
			name:    "YES confirms (case insensitive)",
			input:   "YES\n",
			want:    true,
			wantErr: false,
		},
		{
			name:    "n declines",
			input:   "n\n",
			want:    false,
			wantErr: false,
		},
		{
			name:    "no declines",
			input:   "no\n",
			want:    false,
			wantErr: false,
		},
		{
			name:    "invalid then y confirms",
			input:   "maybe\ny\n",
			want:    true,
			wantErr: false,
		},
		{
			name:    "invalid then n declines",
			input:   "invalid\nn\n",
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout strings.Builder
			stdin := strings.NewReader(tt.input)
			p := NewPrompter(&stdout, stdin)

			got, err := p.Confirm("Proceed?")
			if (err != nil) != tt.wantErr {
				t.Errorf("Confirm() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Confirm() = %v, want %v", got, tt.want)
			}
		})
	}
}
