package template

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var templates embed.FS

// registry holds all parsed templates, initialized on package load.
var registry *template.Template

func init() {
	var err error
	registry, err = parseEmbeddedTemplates()
	if err != nil {
		panic(fmt.Sprintf("template: failed to parse embedded templates: %v", err))
	}
}

// parseEmbeddedTemplates loads and parses all embedded .tmpl files.
func parseEmbeddedTemplates() (*template.Template, error) {
	entries, err := templates.ReadDir("templates")
	if err != nil {
		return nil, fmt.Errorf("read templates dir: %w", err)
	}

	var root *template.Template
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tmpl") {
			continue
		}

		path := filepath.Join("templates", entry.Name())
		content, err := templates.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}

		// Template name is filename without .tmpl extension
		name := strings.TrimSuffix(entry.Name(), ".tmpl")

		if root == nil {
			root = template.New(name)
		} else {
			root = root.New(name)
		}

		if _, err := root.Parse(string(content)); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
	}

	return root, nil
}

// Render executes the named template with the provided data and returns
// the formatted output as a string.
//
// The output is automatically post-processed via Format to normalize
// whitespace and line endings.
//
// Returns an error if the template name is not found or if template
// execution fails. Template syntax errors include line numbers.
//
// Example:
//
//	data := map[string]interface{}{
//	    "Name": "cure",
//	    "Description": "A CLI tool",
//	    "Language": "Go",
//	    "BuildTool": "make",
//	    "TestFramework": "testing",
//	}
//	output, err := template.Render("claude-md", data)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(output)
func Render(name string, data interface{}) (string, error) {
	if registry == nil {
		return "", fmt.Errorf("template registry not initialized")
	}

	tmpl := registry.Lookup(name)
	if tmpl == nil {
		return "", fmt.Errorf("template %q not found (available: %s)", name, strings.Join(List(), ", "))
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template %q: %w", name, err)
	}

	return Format(buf.String()), nil
}

// MustRender is like Render but panics on error.
// Useful for templates that are known to be valid at compile time.
func MustRender(name string, data interface{}) string {
	output, err := Render(name, data)
	if err != nil {
		panic(err)
	}
	return output
}

// List returns the names of all embedded templates.
// Template names are derived from filenames by removing the .tmpl extension.
//
// Example: templates/claude-md.tmpl → "claude-md"
func List() []string {
	if registry == nil {
		return nil
	}

	var names []string
	for _, tmpl := range registry.Templates() {
		if name := tmpl.Name(); name != "" {
			names = append(names, name)
		}
	}
	return names
}

// RenderTo executes the named template and writes output to w.
// Returns the number of bytes written and any error encountered.
//
// Unlike Render, this does not load the entire output into memory,
// making it suitable for large templates or streaming scenarios.
func RenderTo(w io.Writer, name string, data interface{}) (int, error) {
	output, err := Render(name, data)
	if err != nil {
		return 0, err
	}
	return w.Write([]byte(output))
}
