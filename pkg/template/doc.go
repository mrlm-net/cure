// Package template provides a reusable template rendering engine built on
// stdlib text/template with embedded template support.
//
// # Features
//
//   - Embedded templates via //go:embed (no external files required)
//   - Template registry for looking up templates by name
//   - Automatic post-processing (whitespace cleanup, line ending normalization)
//   - Clear error messages with line numbers for syntax errors
//
// # Usage
//
// Render a template with data:
//
//	data := map[string]interface{}{
//	    "Name": "myapp",
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
//
// List available templates:
//
//	names := template.List()
//	fmt.Println("Available templates:", names)
//
// # Template Development
//
// Templates are stored in pkg/template/templates/ with .tmpl extension.
// They are embedded at compile time via //go:embed and parsed during
// package initialization.
//
// Template names are derived from filenames by stripping .tmpl:
//   - templates/claude-md.tmpl → "claude-md"
//   - templates/devcontainer.tmpl → "devcontainer"
//
// # Text Template Syntax
//
// This package uses text/template, which supports:
//
//   - Variables: {{.Name}}
//   - Conditionals: {{if .UseDocker}}...{{end}}
//   - Loops: {{range .Items}}...{{end}}
//   - Comments: {{/* comment */}}
//
// See https://pkg.go.dev/text/template for full syntax reference.
package template
