---
title: "pkg/style"
description: "Minimal ANSI terminal color and style functions with NO_COLOR support"
order: 8
section: "libraries"
---

# pkg/style

`pkg/style` wraps text in ANSI escape codes. All functions are standalone — no struct, no Writer — so they compose naturally and can be used anywhere a string is expected.

**Import path:** `github.com/mrlm-net/cure/pkg/style`

## Colors

```go
import "github.com/mrlm-net/cure/pkg/style"

fmt.Println(style.Red("error"))
fmt.Println(style.Green("ok"))
fmt.Println(style.Yellow("warning"))
fmt.Println(style.Blue("info"))
fmt.Println(style.Magenta("debug"))
fmt.Println(style.Cyan("hint"))
fmt.Println(style.White("text"))
fmt.Println(style.Gray("muted"))
```

## Text styles

```go
fmt.Println(style.Bold("important"))
fmt.Println(style.Dim("subtle"))
fmt.Println(style.Underline("link"))
```

## Composing styles

Functions nest cleanly — wrap calls to combine effects:

```go
fmt.Println(style.Bold(style.Red("fatal error")))
```

The extra reset token from nesting is harmless.

## Stripping codes

```go
plain := style.Reset(style.Bold(style.Red("text")))
// plain == "text"
```

`Reset` removes all ANSI escape sequences from a string using a compiled regular expression. Use it when writing styled output to a file or to a non-terminal.

## NO_COLOR support

`pkg/style` respects the [NO_COLOR](https://no-color.org/) convention. If the `NO_COLOR` environment variable is set at startup, all functions return the original string unchanged.

You can also control styling at runtime:

```go
if !isTerminal {
    style.Disable()
}

style.Enabled() // → false
style.Enable()  // re-enable
```

`Disable` and `Enable` modify package-level state. Commands that write to files or pipes should call `style.Disable()` rather than conditionally wrapping every call in an `if isTerminal` check.
