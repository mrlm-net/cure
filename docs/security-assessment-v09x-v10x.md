# Security Assessment Report — v0.9.x and v0.10.x Epics

**Date:** 2026-03-28
**Assessor:** security-specialist
**Scope:** PRs #139, #140, #141, #142, #143, #144
**Codebase:** github.com/mrlm-net/cure

---

## Scan Summary

| Scanner | Target | Method | Critical | High | Medium | Low | Info |
|---------|--------|--------|----------|------|--------|-----|------|
| Manual code review | All 6 PRs | Source analysis | 0 | 0 | 3 | 4 | 5 |
| OWASP Top 10 checklist | All 6 PRs | Pattern review | 0 | 0 | — | — | — |

No CVEs identified (stdlib-only project; anthropic-sdk-go used on main but not in these PRs).

---

## Findings

### PR #139 — pkg/agent tool foundation (tool.go, message.go, skill.go, agent.go, session.go)

---

#### MEDIUM-1 — A02/A09: Tool input arguments persisted unredacted in session files

**File:** `pkg/agent/message.go` — `ToolUseBlock`
**CWE:** CWE-312 (Cleartext Storage of Sensitive Information)
**OWASP:** A02 Cryptographic Failures, A09 Logging & Monitoring Failures

`ToolUseBlock.Input` is a `map[string]any` that is fully serialized into session JSON files when `MessageContent` marshals tool-use history. If a tool call is passed credentials, tokens, or PII as input arguments (for example, from a skill that wraps an API requiring a bearer token), those values will be persisted verbatim under `~/.local/share/cure/sessions/<id>.json` (mode 0600).

The 0600 permission is a strong mitigation on single-user machines, but does not protect against:
- Backup tools that ignore file permissions (e.g., Time Machine, rsync without `-p`)
- Session files accidentally included in bug reports or exported with `cure context export`
- Multi-user or container environments where processes run under a shared UID

**Recommendation:** Add a comment to `ToolUseBlock` documenting that `Input` is persisted and callers should not pass secret values in tool arguments. Longer term, consider a configurable redaction list (matching key names like `password`, `token`, `secret`, `api_key`) applied at marshal time, similar to how `sanitiseError` operates in the provider adapters.

---

#### LOW-1 — A09: Tool call/result events carry arbitrary string content in event stream

**File:** `pkg/agent/agent.go` — `ToolCallEvent.InputJSON`, `ToolResultEvent.Result`
**CWE:** CWE-532 (Insertion of Sensitive Information into Log File)

`ToolCallEvent.InputJSON` and `ToolResultEvent.Result` are emitted as streaming events and rendered to terminal output or written to NDJSON log files when the user passes `--format ndjson`. Tool results may contain sensitive API responses, file contents, or credentials depending on what tools are registered.

**Recommendation:** Document in the event type comments that callers (REPL, NDJSON formatter) should not assume event content is safe for unredacted persistence. No code change required in this PR; the caller-side formatter should be addressed in a follow-on PR.

---

#### INFO-1 — A08 Data Integrity: Session.Tools correctly excluded from persistence

`Session.Tools []Tool \`json:"-"\`` correctly excludes transient tool state from serialization. Tool interface values hold function pointers that cannot be round-tripped through JSON. This is correct design. No action needed.

---

#### INFO-2 — A03 Injection: MessageContent codec rejects unknown block types

The JSON codec in `message.go` returns an explicit `fmt.Errorf("agent: unknown content block type %q", t.Type)` for unknown discriminator values. This prevents injection of unexpected block types from influencing program behavior. The backward-compatible plain-string path is correctly gated. No action needed.

---

### PR #140 — pkg/mcp Server.Tools() accessor

No security findings. The `Tools()` method returns a copy of the tool slice under `s.mu.RLock()`, preventing data races. The returned slice is independent of the server's internal map. No action needed.

**Security scan: no findings.**

---

### PR #141 — OpenAI adapter (internal/agent/openai/openai.go, internal/agent/sseutil/reader.go)

---

#### MEDIUM-2 — A10 SSRF / A05 Misconfiguration: HTTP client has no timeout, no redirect limit

**File:** `internal/agent/openai/openai.go` line 81
**CWE:** CWE-400 (Uncontrolled Resource Consumption), CWE-601 (Open Redirect)

```go
httpClient: &http.Client{},
```

`http.Client{}` uses Go's default transport with no `Timeout` set. This means:

1. **No total request timeout** — a slow or stalled OpenAI endpoint (or a slow attacker-controlled `OPENAI_BASE_URL`) will block the goroutine indefinitely until the context is cancelled. If the parent context has no deadline, the connection hangs forever.
2. **Unlimited redirects** — the default `http.Client` follows up to 10 redirects. Combined with the `OPENAI_BASE_URL` env override, an attacker who controls that variable (e.g., through environment injection) can cause the client to follow redirects to internal network addresses (SSRF via redirect chain).
3. **Unbounded error response body** — `errBody.ReadFrom(resp.Body)` on line 164 reads the entire error body into a `bytes.Buffer` with no size limit. A malicious server returning a gigabyte error body would cause OOM.

**Recommendation:**
```go
httpClient: &http.Client{
    Timeout: 5 * time.Minute, // covers streaming; context cancellation handles per-request deadline
    CheckRedirect: func(req *http.Request, via []*http.Request) error {
        if len(via) >= 3 {
            return http.ErrUseLastResponse
        }
        return nil
    },
},
```
Replace `errBody.ReadFrom(resp.Body)` with `io.ReadAll(io.LimitReader(resp.Body, 64*1024))`.

---

#### LOW-2 — A05 Misconfiguration: API key appears in Authorization header — sanitiseError is partial mitigation

**File:** `internal/agent/openai/openai.go` — `sanitiseError`

`sanitiseError` performs a `strings.ReplaceAll` of the raw `apiKey` value in error strings. This is a good mitigation but is not exhaustive:
- It only catches exact matches. A 401 error body from the OpenAI API may echo back a partial API key or a different representation.
- It wraps errors with `fmt.Errorf("%s", s)` which loses the original error chain — `errors.Is`/`errors.As` will not traverse through it.

**Recommendation:** The current approach is acceptable for v0.9.x. For the error chain breakage, use `errors.New(s)` instead of `fmt.Errorf("%s", s)` or accept the `goerr113` lint suppression as documented.

---

#### LOW-3 — A09: SSE scanner uses default 64 KB token buffer — potential DoS on extremely long lines

**File:** `internal/agent/sseutil/reader.go`
**CWE:** CWE-400 (Uncontrolled Resource Consumption)

`bufio.NewScanner` defaults to a 64 KB max token size. For normal SSE streams this is sufficient. However, if a malicious or buggy server returns a single line larger than 64 KB, `scanner.Scan()` returns false and `scanner.Err()` returns `bufio.ErrTooLong`, which is propagated as a `parseErr` to the caller and will surface as a user-visible error. This is not a memory safety issue (the scanner will not allocate beyond 64 KB for a single line) but it will cause unexpected session termination.

**Recommendation:** No immediate action required; the behavior is fail-safe (error returned, not OOM). Document in the `Parse` function comment that individual SSE lines are capped at `bufio.MaxScanTokenSize` (64 KB).

---

#### INFO-3 — A02: API key stored in struct field annotated "never emitted in events"

`openaiAdapter.apiKey` is held in the struct only for use by `sanitiseError`. The field is unexported and never included in any `fmt.Sprintf` format string except through `sanitiseError`. No action needed.

---

### PR #142 — Gemini adapter (internal/agent/gemini/gemini.go)

---

#### MEDIUM-3 — A10 SSRF / A02: API key embedded in URL query parameter + unbounded response body

**File:** `internal/agent/gemini/gemini.go` lines 201–202, 311–312
**CWE:** CWE-598 (Sensitive Information in Query String), CWE-400 (Uncontrolled Resource Consumption)

```go
url := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s",
    a.baseURL, a.model, a.apiKey)
```

The Gemini API uses an API key as a URL query parameter (this is the documented Gemini auth pattern, not a custom choice). However, this introduces two risks:

1. **Key in URL** — URLs are logged by proxies, web servers, and network infrastructure. The `sanitiseString` method on the adapter correctly scrubs the key from error messages, but the URL itself may be captured in:
   - OS-level network monitoring tools (Wireshark, tcpdump)
   - Any HTTP proxy in the request path (corporate proxies, local debugging proxies)
   - Go's net/http transport traces if `GODEBUG=http2debug=2` is set

   Mitigation: this is the Gemini API's documented auth mechanism (no alternative header-based auth is available on the `generativelanguage.googleapis.com` endpoint). Document in the adapter that users should not use HTTP inspection tools while the adapter is active.

2. **Unbounded response body on error** — `io.ReadAll(resp.Body)` on line 230 is called on the error body from `streamGenerateContent` with no size limit. A server returning a large error body causes unbounded memory allocation.

   Same pattern on line 331 in `CountTokens`.

**Recommendation:** Replace both `io.ReadAll(resp.Body)` calls with `io.ReadAll(io.LimitReader(resp.Body, 64*1024))`.

The same HTTP client timeout/redirect concerns from MEDIUM-2 apply here identically:
```go
client: &http.Client{},
```
No `Timeout` is set. Same fix applies.

---

#### LOW-4 — A09: Gemini scanner also uses default 64 KB token buffer

**File:** `internal/agent/gemini/gemini.go` line 247

`bufio.NewScanner(resp.Body)` — same analysis as LOW-3 for the OpenAI SSE reader. Fail-safe behavior, but worth documenting.

---

#### INFO-4 — A02: sanitiseString covers both URL-construction error paths

`sanitiseString` is applied to all error message strings that incorporate `a.apiKey` (lines 212, 223, 231, 294). The direct `http.NewRequestWithContext` error on line 213 wraps the URL string, which includes the key — `sanitiseString` correctly scrubs it. No action needed.

---

### PR #143 — Session tags (internal/commands/context/new.go, list.go)

---

#### LOW-5 — A01 Broken Access Control / A03 Injection: Tag values have no validation — future path for injection

**File:** `internal/commands/context/new.go` — `stringSliceFlag.Set`
**CWE:** CWE-20 (Improper Input Validation)

Tag values are rejected only when empty (`v == ""`). There is no maximum length constraint, no character allowlist, and no rejection of control characters or special characters (newlines, null bytes, JSON metacharacters).

Current impact is low because:
- Tags are stored in JSON session files as plain `[]string` values (JSON encoding escapes special characters)
- Tags are displayed in a fixed-width terminal table with truncation at 20 characters
- Tags are used only for exact-match filtering (not passed to any shell or query engine)

However, a tag like `"\n---\n# injected content"` would appear in NDJSON export output as valid JSON (the newline is encoded as `\n`). If a downstream tool parses the NDJSON naively (e.g., reading raw bytes and splitting on `\n`), the embedded newline in a tag value could corrupt the stream.

**Recommendation:** Add a maximum tag length (e.g., 128 characters) and reject tags containing ASCII control characters (bytes 0–31). This is a defense-in-depth measure:

```go
func (f *stringSliceFlag) Set(v string) error {
    if v == "" {
        return fmt.Errorf("tag value cannot be empty")
    }
    if len(v) > 128 {
        return fmt.Errorf("tag value too long (max 128 characters)")
    }
    for _, c := range v {
        if c < 0x20 {
            return fmt.Errorf("tag value contains control character")
        }
    }
    *f = append(*f, v)
    return nil
}
```

---

#### INFO-5 — A01: Tag filter uses exact match — no regex injection risk

The `--tag` filter in `list.go` uses `t == c.tagFilter` (exact string equality). There is no regex evaluation or SQL query construction. No injection risk.

---

### PR #144 — MCP serve command (internal/commands/mcp/serve.go)

---

#### INFO-6 — A05 Misconfiguration: Default HTTP bind address is loopback-only

`WithAddr(c.addr)` defaults to `"127.0.0.1:8080"` (loopback only). The usage message explicitly documents `0.0.0.0:9090` as a non-default example requiring explicit opt-in. The `pkg/mcp` `WithAllowedOrigins` option documents DNS rebinding risk when origins are unrestricted. This is correct defensive design. No action needed.

---

#### INFO-7 — A01: generate tools always use DryRun=true — no filesystem writes from MCP

Both `aiFileOptsFromArgs` and `registerGenerateScaffoldTool` set `DryRun: true` and `NonInteractive: true`. The `writeAIFile` function in `generate/opts.go` routes DryRun=true through the writer path, never calling `fs.AtomicWrite`. MCP tool calls cannot write to the filesystem. No path traversal or TOCTOU risk.

---

#### INFO-8 — A05: MCP server does not validate tool argument string lengths

Tool arguments (`name`, `description`, `language`, `build_tool`, `test_framework`, `conventions`) are passed through `aiFileOptsFromArgs` without length limits. These values flow into Go text templates. The template engine does not execute OS commands, so there is no command injection risk. Very long strings would only affect template rendering time (no OOM risk given Go's template engine). No action required for v0.9.x.

---

## OWASP Top 10 Compliance Summary

| Category | Status | PRs Affected | Notes |
|----------|--------|-------------|-------|
| A01 Broken Access Control | PASS | All | No access control endpoints; session ID allow-list is strict |
| A02 Cryptographic Failures | WARN | #139, #142 | Tool inputs persisted unredacted; API key in Gemini URL (per API design) |
| A03 Injection | PASS | #143 | Tag values have no control-char filter (LOW finding); no exec injection |
| A04 Insecure Design | PASS | All | Panic-on-duplicate is correct for init-time registration |
| A05 Security Misconfiguration | PASS | #141, #142, #144 | HTTP client lacks timeout (MEDIUM); MCP default binds loopback |
| A06 Vulnerable Components | PASS | All | stdlib-only in these PRs; no new external deps introduced |
| A07 Authentication Failures | PASS | All | No auth endpoints introduced |
| A08 Data Integrity | PASS | #139 | json:"-" correctly excludes transient state |
| A09 Logging & Monitoring | WARN | #139, #141, #142 | Tool results/SSE content may carry sensitive data in event streams |
| A10 SSRF | WARN | #141, #142 | No timeout/redirect limits on HTTP clients (MEDIUM finding) |

---

## Consolidated Recommendations

### Immediate (before merge — MEDIUM severity)

1. **PR #141 (OpenAI) and PR #142 (Gemini) — HTTP client hardening:**

   Add a `Timeout` and `CheckRedirect` to both `http.Client{}` initializations:
   ```go
   &http.Client{
       Timeout: 5 * time.Minute,
       CheckRedirect: func(req *http.Request, via []*http.Request) error {
           if len(via) >= 3 {
               return http.ErrUseLastResponse
           }
           return nil
       },
   }
   ```
   This addresses the SSRF-via-redirect risk when `OPENAI_BASE_URL`/`GEMINI_BASE_URL` are overridden.

2. **PR #141 (OpenAI) — bounded error body read:**
   ```go
   // Replace:
   _, _ = errBody.ReadFrom(resp.Body)
   // With:
   const maxErrBody = 64 * 1024
   _, _ = io.Copy(&errBody, io.LimitReader(resp.Body, maxErrBody))
   ```

3. **PR #142 (Gemini) — bounded error body reads (lines 230, 331):**
   ```go
   // Replace both io.ReadAll(resp.Body) with:
   raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
   ```

### Short-term (within 1 sprint — LOW severity)

4. **PR #143 (session tags) — add tag validation:**
   Add max length (128 chars) and control character rejection in `stringSliceFlag.Set`. See LOW-5 for the exact code.

5. **PR #139 (tool blocks) — document persistence of ToolUseBlock.Input:**
   Add a comment to `ToolUseBlock` noting that `Input` is persisted in session files and should not carry sensitive values.

### Long-term (backlog — INFO)

6. Consider a redaction layer for `ToolUseBlock.Input` matching common secret key names before persistence.
7. Document SSE scanner 64 KB line limit in `sseutil.Parse` and `gemini.go` scanner usage.
8. Evaluate whether `cure context export` should strip `ToolUseBlock.Input` values when exporting to Markdown.
