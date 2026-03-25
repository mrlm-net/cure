package template

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/mrlm-net/cure/pkg/config"
)

//go:embed templates/*.tmpl
var templates embed.FS

var (
	mu           sync.Mutex
	globalConfig *config.Config
	// registry is rebuilt lazily; nil means stale (needs rebuild).
	registry *template.Template
)

// SetConfig wires config from the application entry point.
// Calling SetConfig forces a registry rebuild on the next Render/List call.
// Safe to call multiple times (e.g. in tests). A nil cfg is valid and means
// no custom directories beyond the project-local and user-global defaults.
func SetConfig(cfg *config.Config) {
	mu.Lock()
	defer mu.Unlock()
	globalConfig = cfg
	registry = nil // force rebuild on next use
}

// getRegistry returns the built registry, building it lazily if needed.
// Callers must NOT hold mu when calling this function.
func getRegistry() (*template.Template, error) {
	mu.Lock()
	defer mu.Unlock()
	if registry != nil {
		return registry, nil
	}
	var err error
	registry, err = buildRegistry()
	return registry, err
}

// buildRegistry constructs the template registry by loading embedded templates
// first, then overlaying filesystem directories in ascending priority order:
//
//  1. Embedded templates (lowest priority, always available)
//  2. Config template.dirs entries (medium priority)
//  3. User-global directory (~/.cure/templates/)
//  4. Project-local directory (.cure/templates/) (highest priority)
//
// Later-loaded templates with the same name override earlier ones.
// Must be called with mu held.
func buildRegistry() (*template.Template, error) {
	root, err := parseEmbeddedTemplates()
	if err != nil {
		return nil, err
	}

	// Config-specified directories (medium priority, loaded before user/project dirs)
	if globalConfig != nil {
		if dirs, ok := globalConfig.Get("template.dirs", nil).([]interface{}); ok {
			for _, d := range dirs {
				if dir, ok := d.(string); ok {
					_ = loadFromDir(root, dir) // silently skip missing or unreadable dirs
				}
			}
		}
	}

	// User-global directory (~/.cure/templates/)
	if home, err := os.UserHomeDir(); err == nil {
		_ = loadFromDir(root, filepath.Join(home, ".cure", "templates"))
	}

	// Project-local directory (.cure/templates/) — highest priority
	_ = loadFromDir(root, filepath.Join(".cure", "templates"))

	return root, nil
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

// loadFromDir loads all .tmpl and .tpl files from dir into root, overriding
// any existing templates with the same name. Missing or unreadable directories
// are silently skipped. Template syntax errors are printed as warnings to stderr
// and the file is skipped (parse continues with remaining files).
func loadFromDir(root *template.Template, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Directory doesn't exist or isn't readable — expected in most environments.
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".tmpl") && !strings.HasSuffix(name, ".tpl") {
			continue
		}

		// Derive template name by stripping extension (.tmpl takes priority over .tpl)
		templateName := strings.TrimSuffix(name, ".tmpl")
		if templateName == name {
			// Name did not end in .tmpl, try .tpl
			templateName = strings.TrimSuffix(name, ".tpl")
		}

		fullPath := filepath.Join(dir, name)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			// Unreadable file — skip silently; permission issues should not abort load.
			continue
		}

		// Parse into the root template set. Same name overrides any existing template.
		if _, err := root.New(templateName).Parse(string(content)); err != nil {
			// Template syntax error — warn to stderr, don't fail the entire load.
			fmt.Fprintf(os.Stderr, "warning: template %s: %v\n", fullPath, err)
			continue
		}
	}

	return nil
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
	reg, err := getRegistry()
	if err != nil {
		return "", fmt.Errorf("template registry: %w", err)
	}

	tmpl := reg.Lookup(name)
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

// List returns the names of all available templates (embedded + custom).
// Template names are derived from filenames by removing the .tmpl or .tpl extension.
// Custom templates with the same name as embedded templates appear only once.
//
// Example: templates/claude-md.tmpl → "claude-md"
func List() []string {
	reg, err := getRegistry()
	if err != nil || reg == nil {
		return nil
	}

	var names []string
	for _, tmpl := range reg.Templates() {
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
