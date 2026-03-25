---
title: "cure trace"
description: "Trace HTTP, DNS, TCP, and UDP connections with detailed timing"
order: 2
section: "commands"
---

# cure trace

Trace network connections with detailed timing, metadata, and protocol-level events. Output formats include NDJSON for log aggregation and HTML for visual inspection with syntax-highlighted payloads.

## Subcommands

### cure trace dns

Trace DNS resolution with IP addresses, CNAME chain, resolution timing, and RFC 1918 private IP classification.

```sh
cure trace dns api.github.com
cure trace dns api.github.com --server 8.8.8.8
cure trace dns api.github.com --count 5 --interval 500ms
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--format json\|html` | Output format (default: `json`) |
| `--out-file <path>` | Write output to file instead of stdout |
| `--dry-run` | Emit synthetic events without network I/O |
| `--timeout <duration>` | DNS query timeout |
| `--server <ip[:port]>` | DNS server to query (IP address only — hostnames are rejected to avoid DNS bootstrapping circularity) |
| `--count <n>` | Repeat query N times |
| `--interval <duration>` | Delay between repeated queries |

The `--count` and `--interval` flags are useful for detecting intermittent DNS flapping.

### cure trace http

Trace an HTTP request with DNS resolution, TLS handshake, request/response headers, and timing.

```sh
cure trace http https://api.github.com
cure trace http https://api.github.com --format html --output trace.html
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--format json\|html` | Output format (default: `json`) |
| `--output <file>` | Write output to file instead of stdout |
| `--dry-run` | Emit synthetic events without network I/O |

### cure trace tcp

Trace a TCP connection with handshake timing and connection metadata.

```sh
cure trace tcp api.github.com:443
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--format json\|html` | Output format (default: `json`) |
| `--output <file>` | Write output to file instead of stdout |
| `--dry-run` | Emit synthetic events without network I/O |

### cure trace udp

Trace a UDP packet exchange with send/receive timing.

```sh
cure trace udp 8.8.8.8:53
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--format json\|html` | Output format (default: `json`) |
| `--output <file>` | Write output to file instead of stdout |
| `--dry-run` | Emit synthetic events without network I/O |

## Output formats

**NDJSON** — newline-delimited JSON, suitable for log aggregation and processing with tools like `jq`:

```sh
cure trace http https://api.github.com | jq 'select(.event == "response")'
```

**HTML** — rendered report with syntax-highlighted JSON payloads, suitable for sharing or archiving:

```sh
cure trace http https://api.github.com --format html --output report.html
open report.html
```

## Header redaction

The HTTP tracer automatically redacts values for sensitive headers (`Authorization`, `Cookie`, `Set-Cookie`) before emitting events. Redacted values are replaced with `[REDACTED]`.
