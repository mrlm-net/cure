---
title: "pkg/template"
description: "Template generation engine with embedded defaults and custom directory overrides"
order: 10
section: "libraries"
---

# pkg/template

`pkg/template` wraps `text/template` with an embedded template registry and support for user-defined template directories. Templates are loaded lazily on first use and rebuilt whenever configuration changes.

**Import path:** `github.com/mrlm-net/cure/pkg/template`

## Rendering

```go
import "github.com/mrlm-net/cure/pkg/template"

// Render returns the rendered string.
output, err := template.Render("claude-md", data)

// MustRender panics on error — use in tests or init-time setup.
output := template.MustRender("claude-md", data)

// RenderTo writes directly to an io.Writer.
err := template.RenderTo(os.Stdout, "claude-md", data)
```

## Listing available templates

```go
names, err := template.List()
// names is a sorted []string of registered template names.
```

## Custom template directories

`pkg/template` searches four locations in order. The first file with a matching name wins:

1. **Embedded** — templates compiled into the binary via `//go:embed`
2. **Config-defined dirs** — paths from `pkg/config` key `template.dirs`
3. **`~/.cure/templates/`** — user-global overrides
4. **`.cure/templates/`** — project-local overrides (highest precedence)

To enable custom directory resolution, call `SetConfig` after loading your configuration:

```go
cfg, _ := config.Load(...)
template.SetConfig(cfg)
```

Calling `SetConfig` with a new config triggers a lazy rebuild of the registry on the next render call.

### Template file format

Custom templates use `.tmpl` or `.tpl` extensions. The filename without extension becomes the template name:

```
.cure/templates/claude-md.tmpl   → name: "claude-md"
~/.cure/templates/k8sjob.tmpl    → name: "k8sjob"
```

Missing directories are silently skipped. Syntax errors in template files print a warning to stderr and the file is skipped.

## Notes

- The registry is protected by `sync.Mutex` — safe for concurrent renders.
- Template names are case-sensitive.
- Embedded templates are always available as fallbacks even when custom directories are configured.
