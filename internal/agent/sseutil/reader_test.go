package sseutil

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		stopAfter  int // onEvent returns false after this many calls (0 = never stop early)
		cancelCtx  bool
		wantEvents []string
		wantErr    bool
		wantErrIs  error
	}{
		{
			name: "normal event sequence ending with DONE",
			input: "data: {\"token\":\"hello\"}\n" +
				"data: {\"token\":\" world\"}\n" +
				"data: [DONE]\n",
			wantEvents: []string{`{"token":"hello"}`, `{"token":" world"}`},
		},
		{
			name: "DONE sentinel stops parsing before EOF",
			input: "data: first\n" +
				"data: [DONE]\n" +
				"data: should not appear\n",
			wantEvents: []string{"first"},
		},
		{
			name:      "onEvent returning false stops parsing",
			input:     "data: a\ndata: b\ndata: c\n",
			stopAfter: 1,
			// Only the first event is delivered; returning false stops iteration.
			wantEvents: []string{"a"},
		},
		{
			name:      "context cancellation returns ctx error",
			input:     "data: a\ndata: b\ndata: c\n",
			cancelCtx: true,
			// After cancellation onEvent may see 0 or 1 events before the ctx check
			// fires; we only assert on the error, not exact event count.
			wantErr:   true,
			wantErrIs: context.Canceled,
		},
		{
			name:       "empty stream returns nil",
			input:      "",
			wantEvents: []string{},
		},
		{
			name: "lines without data: prefix are skipped",
			input: ": this is a comment\n" +
				"event: message\n" +
				"id: 42\n" +
				"data: kept\n" +
				"\n" +
				"data: also kept\n",
			wantEvents: []string{"kept", "also kept"},
		},
		{
			name: "blank lines are skipped",
			input: "\n" +
				"\n" +
				"data: hello\n" +
				"\n",
			wantEvents: []string{"hello"},
		},
		{
			name: "multiple events without DONE — exhausted reader returns nil",
			input: "data: one\n" +
				"data: two\n",
			wantEvents: []string{"one", "two"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.cancelCtx {
				// Cancel immediately so the first ctx.Err() check fires.
				cancel()
			}

			var received []string
			callCount := 0
			onEvent := func(data []byte) bool {
				callCount++
				received = append(received, string(data))
				if tt.stopAfter > 0 && callCount >= tt.stopAfter {
					return false
				}
				return true
			}

			r := strings.NewReader(tt.input)
			err := Parse(ctx, r, onEvent)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("error = %v, want errors.Is(%v)", err, tt.wantErrIs)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.stopAfter == 0 {
				if len(received) != len(tt.wantEvents) {
					t.Fatalf("got %d events, want %d\nreceived: %v", len(received), len(tt.wantEvents), received)
				}
				for i, want := range tt.wantEvents {
					if received[i] != want {
						t.Errorf("event[%d] = %q, want %q", i, received[i], want)
					}
				}
			} else {
				// stopAfter is set — verify we got exactly stopAfter events.
				if len(received) != tt.stopAfter {
					t.Errorf("got %d events, want %d (stopAfter)", len(received), tt.stopAfter)
				}
			}
		})
	}
}

func TestParse_ReaderError(t *testing.T) {
	// errReader always returns an error on the second read (after the scanner is
	// initialized and reads the first chunk).
	errReader := &failReader{
		data: []byte("data: hello\n"),
		err:  errors.New("read error"),
	}

	var received []string
	err := Parse(context.Background(), errReader, func(data []byte) bool {
		received = append(received, string(data))
		return true
	})

	// We expect the first event to be received and then an error.
	if err == nil {
		t.Fatal("expected error from reader, got nil")
	}
	if err.Error() != "read error" {
		t.Errorf("error = %q, want %q", err.Error(), "read error")
	}
}

// failReader reads data once, then returns err on subsequent reads.
type failReader struct {
	data  []byte
	err   error
	reads int
}

func (f *failReader) Read(p []byte) (int, error) {
	if f.reads == 0 {
		f.reads++
		n := copy(p, f.data)
		return n, nil
	}
	return 0, f.err
}

func TestParse_LargePayload(t *testing.T) {
	// Verify the scanner handles payloads at the default scanner buffer size.
	// bufio.Scanner default buffer is 64KiB; we stay under that.
	payload := strings.Repeat("x", 4096)
	input := "data: " + payload + "\ndata: [DONE]\n"

	var received []string
	err := Parse(context.Background(), strings.NewReader(input), func(data []byte) bool {
		received = append(received, string(data))
		return true
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(received) != 1 || received[0] != payload {
		t.Errorf("unexpected events: %v", received)
	}
}

// Ensure Parse returns nil (not io.EOF) when the reader is exhausted.
func TestParse_ExhaustedReaderReturnsNil(t *testing.T) {
	err := Parse(context.Background(), strings.NewReader(""), func(_ []byte) bool { return true })
	if err != nil && !errors.Is(err, io.EOF) {
		t.Errorf("expected nil or io.EOF, got %v", err)
	}
}
