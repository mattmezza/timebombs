---
name: timebombs-scanning
description: Query the codebase for TIMEBOMB tech-debt annotations using the `timebombs` CLI. Use when the user asks about tech debt, upcoming or past deadlines, exploded bombs, CI `timebombs` failures, or wants to audit debt in a specific area of the repo.
---

# Scanning timebombs

Use this skill when the user asks questions about existing tech debt, or when a `timebombs` CI gate has failed. This skill is **reactive** — do not run scans unprompted.

## Map the user's question to the right command

| User asks… | Run |
|---|---|
| "What's about to blow up?" / "What's due soon?" | `timebombs scan . --within 30d` |
| "What has already exploded?" / "What's past deadline?" | `timebombs scan . --exploded` |
| "How much debt do we have?" | `timebombs scan .` |
| "What will explode by <date>?" | `timebombs scan . --at-time YYYY-MM-DD` |
| "Show me payments-area debt." | `timebombs scan ./src/payments` |
| "Anything from JIRA/FLR/…?" | `timebombs scan . --id JIRA-` |
| "Only the debt in this PR." | `timebombs scan . --changed-only --base origin/main` |
| "Give me structured output." | `timebombs scan . --format json` |
| "For the security scanner / code scanning." | `timebombs scan . --format sarif` |

## Flag combining

Filters stack. Useful combinations:

- `--exploded --id FLR-` — exploded bombs owned by FlowRent.
- `--within 14d --ticking` — ticking only, next two weeks.
- `--changed-only --max-exploded 0` — PR gate: no new exploded bombs.

## CI failure playbook

Exit code `1` from `timebombs scan --max-exploded N` = more than `N` bombs are past their deadline.

1. Run `timebombs scan . --exploded` to list the offenders.
2. For each bomb, there are exactly two honest resolutions:
   - **Do the work.** The deadline was set for a reason.
   - **Renegotiate the deadline in a PR.** Bumping the date in a diff makes the trade-off visible to the team.
3. **Do not** suppress the check (`|| true`, commenting out the step, bumping `--max-exploded`). Hidden debt is worse than visible debt.

## When asked "should I make this a timebomb?"

If the deadline is real and the consequence is concrete → yes. Route to the **timebombs-planting** skill.

If it's aspirational or blocked indefinitely → a plain `TODO` is the honest choice. Not everything is a timebomb.

## When the CLI is not installed

Install via:

```bash
go install github.com/mattmezza/timebombs/cmd/timebombs@latest
```

Or grab a release binary from https://github.com/mattmezza/timebombs/releases.
