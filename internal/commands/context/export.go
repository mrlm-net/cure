package ctxcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/fs"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// exportMarkdownTmpl is the Go text/template used to render a session as Markdown.
// It is embedded directly in this file to keep the export command self-contained
// and independent of the pkg/template registry.
const exportMarkdownTmpl = `# {{ .ID }}

| Field    | Value |
|----------|-------|
| Provider | {{ .Provider }} |
| Model    | {{ .Model }} |
| Created  | {{ formatTime .CreatedAt }} |
| Updated  | {{ formatTime .UpdatedAt }} |{{ if .ForkOf }}
| Fork of  | {{ .ForkOf }} |{{ end }}

---
{{ if .History }}{{ range .History }}
## {{ titleCase .Role }}

{{ .Content }}
{{ end }}{{ else }}
_No messages in this session._
{{ end }}`

// ExportCommand implements "cure context export <session-id>".
// It renders a saved session as Markdown (default) or raw NDJSON and writes
// the result to stdout or an optional output file. It is read-only and never
// mutates the session in the store.
type ExportCommand struct {
	store agent.SessionStore

	// Flags
	format string
	output string
}

func (c *ExportCommand) Name() string        { return "export" }
func (c *ExportCommand) Description() string { return "Export a session to Markdown or NDJSON" }

func (c *ExportCommand) Usage() string {
	return `Usage: cure context export <session-id> [flags]

Exports a saved session to Markdown (default) or NDJSON format.

Note: flags must be supplied before the positional <session-id> argument
because Go's flag package stops parsing flags at the first non-flag token.

Arguments:
  <session-id>    ID of the session to export (required)

Flags:
  --format  Output format: "markdown" (default) or "ndjson"
  --output  Write to file instead of stdout

Examples:
  cure context export abc123
  cure context export abc123 --format ndjson
  cure context export --output session.md abc123
`
}

func (c *ExportCommand) Flags() *flag.FlagSet {
	fset := flag.NewFlagSet("context-export", flag.ContinueOnError)
	fset.StringVar(&c.format, "format", "markdown", `Output format: "markdown" or "ndjson"`)
	fset.StringVar(&c.output, "output", "", "Write to file path instead of stdout")
	return fset
}

// Run implements terminal.Command for ExportCommand.
func (c *ExportCommand) Run(ctx context.Context, tc *terminal.Context) error {
	if len(tc.Args) == 0 {
		fmt.Fprintln(tc.Stderr, c.Usage())
		return fmt.Errorf("context export: missing required <session-id> argument")
	}
	id := tc.Args[0]

	s, err := c.store.Load(ctx, id)
	if err != nil {
		if errors.Is(err, agent.ErrSessionNotFound) {
			return fmt.Errorf("context export: session not found: %s", id)
		}
		return fmt.Errorf("context export: load: %w", err)
	}

	var content []byte
	switch c.format {
	case "markdown", "":
		content, err = renderMarkdown(s)
	case "ndjson":
		content, err = renderNDJSON(s)
	default:
		return fmt.Errorf("context export: unsupported format %q; expected \"markdown\" or \"ndjson\"", c.format)
	}
	if err != nil {
		return fmt.Errorf("context export: render: %w", err)
	}

	if c.output == "" {
		_, err = tc.Stdout.Write(content)
		return err
	}

	// Ensure parent directory exists before writing.
	if dir := filepath.Dir(c.output); dir != "." {
		if err := fs.EnsureDir(dir, 0755); err != nil {
			return fmt.Errorf("context export: create dir: %w", err)
		}
	}
	return fs.AtomicWrite(c.output, content, 0644)
}

// renderMarkdown renders the session as a Markdown document using
// exportMarkdownTmpl. The result contains an H1 heading with the session ID,
// a metadata table, and one H2 section per message in the history.
func renderMarkdown(s *agent.Session) ([]byte, error) {
	funcMap := template.FuncMap{
		// formatTime formats a time.Time value as "2006-01-02 15:04:05 UTC".
		// The parameter is typed as interface{ Format(string) string } so the
		// template function works with any time-like value without importing
		// the time package.
		"formatTime": func(t interface{ Format(string) string }) string {
			return t.Format("2006-01-02 15:04:05 UTC")
		},
		// titleCase upper-cases the first character of the role name so that
		// "user" renders as "User" and "assistant" renders as "Assistant".
		// Accepts agent.Role (a named string type) to avoid template type mismatch.
		"titleCase": func(r agent.Role) string {
			s := string(r)
			if len(s) == 0 {
				return s
			}
			return strings.ToUpper(s[:1]) + s[1:]
		},
	}
	tmpl, err := template.New("export").Funcs(funcMap).Parse(exportMarkdownTmpl)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, s); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// renderNDJSON serialises the session as a single pretty-printed JSON object
// followed by a newline. Using json.Encoder ensures the output is terminated
// with a newline, consistent with the NDJSON convention used elsewhere.
func renderNDJSON(s *agent.Session) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
