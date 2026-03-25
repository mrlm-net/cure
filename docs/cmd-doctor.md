---
title: "cure doctor"
description: "Project health check — validates README, tests, CI, .gitignore, CLAUDE.md, build tool, and dependency manifest"
order: 3
section: "commands"
---

# cure doctor

`cure doctor` runs a set of project health checks and reports the result for each one. It exits with code `1` if any check fails, making it safe to use in CI pipelines.

## Usage

```sh
cure doctor
```

No flags are required. The command inspects the current working directory.

## Checks

| Check | Pass condition | Level |
|-------|----------------|-------|
| README | `README.md` exists | fail |
| Tests | `*_test.go` files found | fail |
| CI | `.github/workflows/` or `.gitlab-ci.yml` found | fail |
| .gitignore | `.gitignore` exists | warn |
| CLAUDE.md | `CLAUDE.md` exists | fail |
| Build tool | `Makefile`, `Taskfile.yml`, or `justfile` found | fail |
| Dependency manifest | `go.mod`, `package.json`, `pyproject.toml`, `Cargo.toml`, or `requirements.txt` found | fail |

**warn** checks print a warning but do not affect the exit code. **fail** checks cause `cure doctor` to exit `1`.

## Example output

```
cure doctor

  ✓  README
  ✓  Tests
  ✓  CI
  ⚠  .gitignore   — missing; consider adding one
  ✓  CLAUDE.md
  ✓  Build tool
  ✓  Dependency manifest

1 warning
```

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | All checks passed (warnings are allowed) |
| `1` | One or more checks failed |

## Extending

`cure doctor` is built on the `CheckFunc` type — `func() CheckResult` — following the `http.HandlerFunc` pattern. Additional checks can be added by implementing the type and registering them in `internal/commands/doctor/doctor.go`.
