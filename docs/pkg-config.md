---
title: "pkg/config"
description: "Hierarchical configuration merging with dot-notation access"
order: 5
section: "libraries"
---

# pkg/config

`pkg/config` provides hierarchical configuration management with deep merging, dot-notation access, and multiple loaders. It is designed to implement the standard precedence chain used by cure: defaults < global file < local file < environment variables < CLI flags.

**Import path:** `github.com/mrlm-net/cure/pkg/config`

## ConfigObject

`ConfigObject` is a `map[string]interface{}` with helper methods for deep merging and dot-notation access:

```go
cfg := config.ConfigObject{
    "database": map[string]interface{}{
        "host": "localhost",
        "port": 5432,
    },
}

// Dot-notation get
host := cfg.Get("database.host") // "localhost"

// Dot-notation set
cfg.Set("database.port", 5433)
```

## DeepMerge

`DeepMerge` combines two `ConfigObject` values. Nested maps are merged recursively; scalar values in the source override those in the destination:

```go
base := config.ConfigObject{"debug": false, "server": map[string]interface{}{"port": 8080}}
override := config.ConfigObject{"server": map[string]interface{}{"port": 9090}}

merged := config.DeepMerge(base, override)
// merged["server"]["port"] == 9090
// merged["debug"] == false (preserved from base)
```

## Loaders

### JSONFile loader

Loads configuration from a JSON file with tilde expansion in the path:

```go
global, err := config.JSONFile("~/.cure.json")
local, err := config.JSONFile(".cure.json")
```

### Environment loader

Loads configuration from environment variables matching a given prefix. The `CURE_` prefix maps to dot-notation keys:

```go
// CURE_DATABASE_HOST=myhost → {"database": {"host": "myhost"}}
env := config.Environment("CURE_")
```

## Precedence chain

Cure loads configuration in this order (later sources win):

1. Defaults (hardcoded in the binary)
2. Global config: `~/.cure.json`
3. Local config: `.cure.json` in the current directory
4. Environment variables (`CURE_` prefix)
5. CLI flags

```go
cfg := config.ConfigObject{}
cfg = config.DeepMerge(cfg, defaults)
cfg = config.DeepMerge(cfg, globalFile)
cfg = config.DeepMerge(cfg, localFile)
cfg = config.DeepMerge(cfg, envVars)
cfg = config.DeepMerge(cfg, cliFlags)
```

The merged config is passed to commands via `terminal.Context.Config`.
