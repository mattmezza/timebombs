# timebombs v2 — Project Specification

## What this is

`timebombs` is a static analysis CLI tool that scans codebases for structured comment annotations representing conscious technical debt with deadlines. It is language-agnostic, dependency-free for adopters, and designed to integrate into CI pipelines and AI coding agent workflows.

The core thesis: in the era of AI coding agents shipping code at unprecedented velocity, teams need a lightweight, enforceable mechanism to track and pay back tech debt. A timebomb is a TODO with a deadline — and teeth.

## Philosophy

- **Zero adoption friction.** A timebomb is a comment. No imports, no libraries, no runtime cost.
- **The annotation is the developer's intent. The CLI is the team's policy.** Keep these concerns separate.
- **Ship less, ship right.** No config files, no SaaS dashboard, no plugins. Flags, exit codes, stdout. Unix philosophy.

---

## Annotation Format

### Syntax

```
TIMEBOMB(<deadline>[, <id>]): <description>
```

- `deadline` — ISO 8601 date (`YYYY-MM-DD`). **Required.** Always the first argument.
- `id` — Free-form string, typically a ticket reference (e.g., `JIRA-123`, `FLR-42`, `#317`). **Optional.**
- `description` — Human-readable explanation of the debt and what needs to happen. **Required.**

### Multi-line descriptions

The `TIMEBOMB(...)` marker starts the annotation. Any immediately following comment lines that are indented (relative to the marker line) are continuation lines belonging to the same bomb. A non-indented comment line, a blank line, or a non-comment line ends the annotation.

### Examples across languages

```python
# TIMEBOMB(2025-09-01): Remove v1 endpoints after migration complete.
#   The new v2 endpoints are already serving 90% of traffic.
#   Blocked by: mobile app rollout to force-update v1 clients.
```

```typescript
// TIMEBOMB(2025-09-01, JIRA-123): Replace polling with WebSocket.
//   This was a quick fix for the demo. The polling interval is 5s
//   which puts unnecessary load on the API under high concurrency.
```

```go
/* TIMEBOMB(2025-11-15): Rip out the feature flag scaffolding.
   We shipped the experiment, the flag is always-on now, but there's
   still branching logic everywhere in the checkout flow. */
```

```ruby
# TIMEBOMB(2025-08-01, #317): Remove this unsafe workaround.
```

```sql
-- TIMEBOMB(2025-10-01): Drop the legacy_users table.
--   All data has been migrated to the new schema.
```

### Comment style support

The scanner must recognize these comment prefixes: `//`, `#`, `--`, `/*...*/`, `{-...-}`, `%`, `;;`, `rem`, `'` (VB). The parser strips the comment prefix before matching the `TIMEBOMB(` pattern.

For block comments (`/* */`, `{- -}`), continuation lines may be prefixed with `*` or whitespace only (no repeated comment marker). The block comment closing token ends the annotation.

### What is NOT a timebomb

- `TODO`, `FIXME`, `HACK`, `XXX` — these are not timebombs. The scanner ignores them entirely.
- A comment containing the word "timebomb" without the `TIMEBOMB(` pattern is not a timebomb.

---

## Bomb States

A timebomb has exactly two states:

- **Ticking** — the deadline is in the future (relative to "now").
- **Exploded** — the deadline is in the past or is today.

There is no "armed" state. A planted timebomb is inherently armed. The distinction between "about to explode" and "not yet concerning" is a query-time filter (`--within`), not a bomb property.

---

## CLI Design

### Invocation

```
timebombs [command] [path] [flags]
```

`path` defaults to `.` (current directory).

### Commands

#### `scan` (default command)

Scan the codebase for timebomb annotations and report their status.

```bash
timebombs scan .                           # scan current directory
timebombs .                                # same (scan is the default command)
timebombs scan ./src ./lib                 # scan multiple paths
```

**Output control:**

| Flag | Values | Default | Description |
|------|--------|---------|-------------|
| `--format` | `text`, `json`, `sarif` | `text` | Output format |
| `--quiet` | — | off | Suppress all output, use exit code only |

**Filtering:**

| Flag | Example | Description |
|------|---------|-------------|
| `--within <duration>` | `--within 30d`, `--within 2w` | Only show bombs exploding within this window from "now" |
| `--at-time <iso-date>` | `--at-time 2025-10-01` | Pretend "now" is this date (for sprint planning) |
| `--id <prefix>` | `--id JIRA-`, `--id FLR-` | Filter by ID prefix |
| `--exploded` | — | Only show exploded bombs |
| `--ticking` | — | Only show ticking bombs |

Filters are combinable. `--exploded --id FLR-` shows all FlowRent-tagged exploded bombs.

**CI gating:**

| Flag | Example | Description |
|------|---------|-------------|
| `--max-exploded <n>` | `--max-exploded 0` | Exit non-zero if more than n bombs have exploded |

**Scanner behavior:**

| Flag | Description |
|------|-------------|
| `--exclude <glob>` | Exclude paths (repeatable). Respects `.gitignore` by default. |
| `--no-gitignore` | Don't auto-respect `.gitignore` |

**Exit codes:**

| Code | Meaning |
|------|---------|
| `0` | Success (all thresholds met) |
| `1` | Threshold exceeded |
| `2` | Usage/parse error |

#### `init`

Install agent skills and optionally a CI workflow.

```bash
timebombs init                        # auto-detect agents in the repo
timebombs init --agent claude-code    # install for a specific agent
timebombs init --all                  # install for all known agents
timebombs init --list                 # list supported agents and their file paths
timebombs init --ci github-actions    # also generate CI workflow file
```

**Agent detection and installation targets:**

| Agent | Detection signal | Install target |
|-------|------------------|----------------|
| Claude Code | `.claude/` dir or `CLAUDE.md` exists | `.claude/timebombs.md` |
| Codex CLI | `codex/` dir or `AGENTS.md` exists | Append to `codex/AGENTS.md` |
| OpenCode | `.opencode/` dir exists | `.opencode/agents/timebombs.md` |
| Cursor | `.cursor/` dir or `.cursorrules` exists | Append to `.cursor/rules` |
| GitHub Copilot | `.github/` dir exists | Append to `.github/copilot-instructions.md` |

For agents that use a single shared file, append the skill content with a clear delimiter (e.g., a markdown heading `## Timebombs`). For agents that support multiple skill files, create a dedicated file.

`init` should be idempotent — running it twice doesn't duplicate content.

**CI workflow generation:**

| Flag | Values | Description |
|------|--------|-------------|
| `--ci` | `github-actions` | Generate CI workflow file |

For `github-actions`, generate `.github/workflows/timebombs.yml` that runs on PR and push to main, executes `timebombs scan . --max-exploded 0 --format text`, and optionally uploads SARIF.

#### `version`

Print version and exit.

```bash
timebombs version
```

### Text output format

The default `text` output should be opinionated and pretty, inspired by `ripgrep` and `fd`:

- Group results by file path.
- Color-code deadlines: red for exploded, yellow for within 30 days, dim/default for distant.
- Show a summary line at the end: `N timebombs: X ticking, Y exploded`.
- When used in a non-TTY context (piped), strip colors automatically.

Example:

```
src/api/v1/endpoints.py
  L42  2025-05-22  JIRA-123  Remove v1 endpoints after migration complete.  [EXPLODED]

src/payments/checkout.go
  L118  2025-11-15            Rip out the feature flag scaffolding.          [ticking, 215d]

src/auth/oauth.ts
  L7   2025-08-01  #317      Remove this unsafe workaround.                 [ticking, 110d]

3 timebombs: 2 ticking, 1 exploded
```

### JSON output format

```json
{
  "scanned_at": "2026-04-13",
  "summary": {
    "total": 3,
    "ticking": 2,
    "exploded": 1
  },
  "timebombs": [
    {
      "file": "src/api/v1/endpoints.py",
      "line": 42,
      "deadline": "2025-05-22",
      "id": "JIRA-123",
      "description": "Remove v1 endpoints after migration complete.\nThe new v2 endpoints are already serving 90% of traffic.\nBlocked by: mobile app rollout to force-update v1 clients.",
      "state": "exploded",
      "days_remaining": -327
    }
  ]
}
```

`days_remaining` is negative when exploded, positive when ticking.

### SARIF output format

Standard SARIF 2.1.0. Each timebomb is a `result` with:
- `level`: `error` for exploded, `warning` for ticking.
- `message.text`: the bomb description.
- `locations`: file path and line number.
- `properties`: deadline, id, state, days_remaining.

---

## Agent Skills

### Planting Skill

The planting skill teaches AI coding agents when and how to annotate tech debt with timebombs. This is the growth engine — it works without the CLI being installed.

**Core instructions the skill must convey:**

1. **When to plant a timebomb:**
   - Writing a workaround or shortcut.
   - Adding a backward-compatibility shim.
   - Hardcoding configuration values.
   - Implementing a naive or unoptimized solution (N+1 queries, no pagination, no caching).
   - Pinning a dependency version to work around a bug.
   - Leaving a feature flag in after a rollout.
   - Adding temporary error handling (broad catches, swallowed errors).
   - Keeping dead code paths alive during a migration.

2. **When NOT to plant a timebomb:**
   - Not every TODO is a timebomb. If there's no meaningful deadline, use a regular comment.
   - Don't bomb aspirational improvements ("it would be nice to...").
   - Don't bomb things that are blocked on external factors with no expected resolution date.

3. **How to pick a deadline:**
   - Default to 90 days from today if unsure.
   - Align with known milestones (end of sprint, release date, migration deadline).
   - Shorter fuse = higher urgency. The deadline IS the severity.

4. **How to write a good description:**
   - State WHAT needs to happen (the action).
   - State WHY it was deferred (the context).
   - State WHAT breaks or degrades if it's not done (the consequence).
   - Multi-line is fine. Be specific.

5. **Exact format:**
   ```
   TIMEBOMB(<YYYY-MM-DD>[, <id>]): <description>
   ```

### Scanning Skill

The scanning skill teaches AI coding agents how to use the `timebombs` CLI for reporting and planning.

**Core instructions:**

1. **Common queries and their flag translations:**
   - "What's about to blow up?" → `timebombs scan . --within 30d`
   - "What has already exploded?" → `timebombs scan . --exploded`
   - "How much debt do we have?" → `timebombs scan .`
   - "What will explode by next month?" → `timebombs scan . --at-time <date>`
   - "Show me all payments-related debt" → `timebombs scan ./src/payments`
   - "Give me the debt as JSON" → `timebombs scan . --format json`

2. **Interpreting CI failures:**
   - Exit code 1 means a `--max-exploded` threshold was exceeded.
   - Run `timebombs scan . --exploded` to see which bombs failed the gate.
   - Resolution: either do the work, or bump the deadline in a PR (making the debt renegotiation visible).

---

## Tech Stack

- **Language:** Go
- **Target platforms:** Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64)
- **Build:** Standard `go build`. Use `goreleaser` for cross-compilation and release automation.
- **Dependencies:** Minimize. Prefer stdlib where reasonable. Acceptable dependencies:
  - CLI framework (e.g., `cobra` or `pflag`, or just stdlib `flag`)
  - Gitignore matching (e.g., `go-gitignore` or `doublestar`)
  - Terminal colors (e.g., `fatih/color` or `lipgloss`)
- **Testing:** Standard `go test`. Table-driven tests for the parser. Integration tests with fixture directories containing annotated files in various languages.
- **Linting:** `golangci-lint` with a reasonable config.

### Project structure

```
timebombs/
├── cmd/
│   └── timebombs/
│       └── main.go              # Entry point
├── internal/
│   ├── scanner/
│   │   ├── scanner.go           # File walker + orchestration
│   │   ├── scanner_test.go
│   │   ├── parser.go            # Comment detection + TIMEBOMB annotation parsing
│   │   └── parser_test.go
│   ├── model/
│   │   └── timebomb.go          # Timebomb struct and state logic
│   ├── output/
│   │   ├── text.go              # Pretty terminal output
│   │   ├── json.go              # JSON formatter
│   │   └── sarif.go             # SARIF formatter
│   └── init/
│       ├── detect.go            # Agent detection
│       ├── install.go           # Skill installation
│       └── templates/           # Embedded skill templates per agent
│           ├── claude-code.md
│           ├── codex.md
│           ├── opencode.md
│           ├── cursor.md
│           ├── copilot.md
│           └── ci/
│               └── github-actions.yml
├── testdata/                    # Fixture files for integration tests
│   ├── python/
│   ├── typescript/
│   ├── go/
│   ├── ruby/
│   ├── sql/
│   └── mixed/
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── LICENSE
└── .goreleaser.yml
```

Templates in `internal/init/templates/` should be embedded using Go's `embed` package so the binary is self-contained.

---

## Implementation Order

Build in this sequence. Each step should be a working, testable increment.

### Phase 1: Core scanner

1. **Model** — `Timebomb` struct with fields: `File`, `Line`, `Deadline`, `ID`, `Description`, `State` (computed from deadline vs now).
2. **Parser** — Given a line (with comment prefix already stripped), detect `TIMEBOMB(` pattern, extract deadline, optional ID, description. Handle multi-line by accepting a slice of continuation lines.
3. **Scanner** — Walk directory tree, respect `.gitignore`, read files line by line, detect comment prefixes, feed to parser, accumulate results.
4. **Text output** — Pretty-print results grouped by file with colored status.
5. **Wire it up** — `main.go` with `scan` command, path argument, basic flags (`--format text`, `--exploded`, `--ticking`, `--within`, `--at-time`, `--id`, `--max-exploded`, `--exclude`, `--no-gitignore`, `--quiet`).

### Phase 2: Additional output formats

6. **JSON output** — `--format json`.
7. **SARIF output** — `--format sarif`.

### Phase 3: Init command

8. **Agent detection** — Look for known directories/files to identify which agents are in use.
9. **Skill templates** — Write the planting skill content for each supported agent. Embed with `//go:embed`.
10. **Init command** — `timebombs init` with `--agent`, `--all`, `--list`, `--ci` flags.

### Phase 4: Release

11. **goreleaser config** — Cross-compile for all target platforms, generate checksums, publish to GitHub Releases.
12. **GitHub Action** — Create a reusable action (`mattmezza/timebombs-action`) that downloads the binary and runs the scan.
13. **README** — Clear, concise, with the "AI writes your code, who cleans it up?" angle. Installation, quickstart, CI setup, link to docs.

---

## Testing Strategy

- **Parser unit tests** — Table-driven. Cover all comment styles, single-line, multi-line, edge cases (malformed dates, missing description, empty ID, no closing paren, nested parens in description).
- **Scanner integration tests** — Fixture directories in `testdata/` with known bombs. Assert correct count, state, line numbers.
- **Output tests** — Snapshot tests for JSON and SARIF (compare against golden files). For text output, assert key content presence (don't snapshot colors).
- **Init tests** — Temp directories with various agent configs. Assert correct file creation/appending, idempotency.
- **CLI integration tests** — Run the built binary against fixture directories, assert exit codes and stdout.

---

## Non-goals (explicitly out of scope for v1)

- Runtime library in any language.
- Config file (`.timebombsrc`, `timebombs.toml`, etc.).
- Web dashboard or SaaS component.
- Language-specific AST parsing (we parse comments, not code).
- Webhook/notification sending from the CLI (pipe JSON to `curl`).
- Plugin system.
- IDE extensions.
