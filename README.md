# timebombs

> AI writes your code. Who cleans it up?

`timebombs` is a language-agnostic static analysis CLI that scans codebases for tech debt *with deadlines*. A timebomb is a structured comment — no library to import, no runtime cost, zero adoption friction.

```
TIMEBOMB(<YYYY-MM-DD>[, <id>]): <description>
```

In the era of AI coding agents shipping at unprecedented velocity, teams need a lightweight, enforceable mechanism to track and pay back the debt that agents (and humans) create. `timebombs` is a TODO with teeth.

## Install

### One-liner (Linux / macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/mattmezza/timebombs/main/install.sh | bash
```

Auto-detects OS/arch and drops the binary in `/usr/local/bin` (falling back to `~/.local/bin` if the system dir isn't writable). Override with env vars:

```bash
curl -fsSL https://raw.githubusercontent.com/mattmezza/timebombs/main/install.sh | VERSION=v0.4.0 bash
curl -fsSL https://raw.githubusercontent.com/mattmezza/timebombs/main/install.sh | INSTALL_DIR=$HOME/bin bash
```

### Binary releases

Grab the latest binary for your platform from the [releases page](https://github.com/mattmezza/timebombs/releases).

### Go

```bash
go install github.com/mattmezza/timebombs/cmd/timebombs@latest
```

## Quickstart

Annotate your code:

```python
# TIMEBOMB(2025-09-01, JIRA-123): Remove v1 endpoints after migration.
#   v2 now serves 90% of traffic; blocked on mobile force-update.
```

```typescript
// TIMEBOMB(2025-12-31): Replace polling with WebSocket.
```

Scan:

```bash
$ timebombs scan .

src/api/v1/endpoints.py
  L42  2025-09-01  JIRA-123  Remove v1 endpoints after migration.  [EXPLODED]

src/poller.ts
  L7   2025-12-31            Replace polling with WebSocket.       [ticking, 262d]

2 timebombs: 1 ticking, 1 exploded
```

## Common queries

| Question | Command |
|---|---|
| What's about to blow up? | `timebombs scan . --within 30d` |
| What has exploded? | `timebombs scan . --exploded` |
| How much debt total? | `timebombs scan .` |
| What will explode by a date? | `timebombs scan . --at-time 2026-06-01` |
| Show a specific area. | `timebombs scan ./src/payments` |
| Filter by ticket prefix. | `timebombs scan . --id JIRA-` |
| Give me JSON. | `timebombs scan . --format json` |
| Give me SARIF. | `timebombs scan . --format sarif` |
| Only scan what changed in this PR. | `timebombs scan . --changed-only --base origin/main` |
| Allowlist certain files. | `timebombs scan . --include '**/*.go'` |
| Scan one file from a hook. | `cat file.py \| timebombs scan --stdin --stdin-filename file.py` |

## CI integration

### GitHub Actions (generated)

```bash
timebombs init --ci github-actions
```

Drops `.github/workflows/timebombs.yml` that fails the build on any exploded bomb and uploads a SARIF report to code scanning.

### GitHub Actions (reusable action)

```yaml
- uses: mattmezza/timebombs@v1
  with:
    path: .
    max-exploded: 0
    upload-sarif: true
```

### Any CI system

```bash
timebombs scan . --max-exploded 0
```

Exit codes:
- `0` — all clear.
- `1` — threshold exceeded.
- `2` — usage error.

## AI agent integration

`timebombs init` installs a "Timebombs" skill for the coding agents present in your repo, teaching them when and how to plant bombs — and how to run the CLI to reason about them.

```bash
timebombs init                      # auto-detect agents
timebombs init --all                # install for every supported agent
timebombs init --agent claude-code  # specific agent
timebombs init --list               # show what's supported
```

Supported: Claude Code, Codex CLI, OpenCode, Cursor, GitHub Copilot.

## Annotation format

```
TIMEBOMB(<deadline>[, <id>]): <description>
```

- **deadline** — ISO 8601 date. Required. First argument.
- **id** — ticket reference like `JIRA-123`, `#317`. Optional.
- **description** — what needs to happen, why it was deferred, what breaks if it isn't. Required.

### Multi-line descriptions

```python
# TIMEBOMB(2025-09-01): Remove v1 endpoints after migration.
#   v2 now serves 90% of traffic.
#   Blocked by: mobile force-update rollout.
```

Continuation lines must be **indented past** the marker (for line comments) or **inside the same block** (for `/* */`, `{- -}`).

### Comment styles supported

`//`, `#`, `--`, `/* */`, `{- -}`, `%`, `;;`, `rem`, `'` (VB).

## When to plant a timebomb

- Workarounds, shortcuts, quick demo fixes.
- Backward-compatibility shims during migrations.
- Hardcoded values you *know* need to be configurable.
- Naive implementations (N+1, no pagination, no caching).
- Dependency pins that work around an upstream bug.
- Feature flags that should be removed after rollout.
- Broad error swallowing.
- Dead code kept alive for a transition.

**Don't** bomb aspirational improvements or debt with no realistic deadline. Use a plain `TODO` for those.

## Philosophy

- **Zero adoption friction.** The annotation is a comment.
- **The annotation is intent. The CLI is policy.** Keep them separate.
- **Ship less, ship right.** No config files, no SaaS, no plugins. Flags, exit codes, stdout.

## License

MIT — see [LICENSE](LICENSE).
