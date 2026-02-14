package formatter

import (
	"encoding/json"
	"io"

	"github.com/mrlm-net/cure/pkg/tracer/event"
)

// NDJSONEmitter writes events as newline-delimited JSON.
type NDJSONEmitter struct {
	w io.Writer
}

// NewNDJSONEmitter creates an emitter that writes NDJSON to w.
func NewNDJSONEmitter(w io.Writer) *NDJSONEmitter {
	return &NDJSONEmitter{w: w}
}

// Emit writes a single event as a JSON line.
func (e *NDJSONEmitter) Emit(ev event.Event) error {
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = e.w.Write(append(data, '\n'))
	return err
}

// Close is a no-op for NDJSON (implements event.Emitter).
func (e *NDJSONEmitter) Close() error {
	return nil
}
