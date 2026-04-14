## Timebombs

`timebombs` tracks conscious tech debt with deadlines via structured comment annotations. Two jobs here: **planting** them when writing debt, and **scanning** them when asked about debt.

### When to plant (applies whenever you write code)

IF you write any of these, THEN plant a `TIMEBOMB` in the same edit:

- A workaround or quick fix (demo deadline, fire drill, "just ship it").
- A backward-compat shim, adapter, or wrapper during a migration.
- A hardcoded value that belongs in config.
- A naive implementation: N+1, missing pagination, no caching, linear scan over something that will grow.
- A feature flag after the experiment shipped.
- A dependency pin that works around an upstream bug.
- Broad error catching, swallowed errors.
- Dead code kept alive during a transition.

**Do NOT** plant for aspirational improvements or debt with no realistic deadline. Use a plain `TODO` there — a fake deadline is worse than no deadline.

### Format

```
TIMEBOMB(<YYYY-MM-DD>[, <id>]): <description>
```

- **deadline** — ISO 8601, required, first argument.
- **id** — ticket like `JIRA-123`, `#317`. Optional but recommended when a ticket exists.
- **description** — state WHAT must happen, WHY it was deferred, WHAT BREAKS if it is not done.

Continuation lines: indent past the marker (line comments), or stay inside the same block (`/* */`, `{- -}`).

### Picking a deadline

- Default: **90 days** from today if no anchor.
- Anchor to a known milestone (sprint end, release, migration cutover).
- Shorter fuse = higher urgency. The deadline IS the severity.

### Examples

```typescript
// TIMEBOMB(2026-07-15, FLR-42): Replace polling with the WS channel.
//   Shipped 5s polling to hit the demo. Under load this saturates
//   /api/dashboard. Break once WS auth (FLR-41) lands.
setInterval(refreshDashboard, 5000);
```

```python
# TIMEBOMB(2026-06-01): Move rate limit to runtime config.
#   Hardcoded 100 rps to ship tomorrow. Ops cannot tune without a deploy.
RATE_LIMIT = 100
```

```go
/* TIMEBOMB(2026-09-30, MIG-17): Drop v1 response wrapper.
   Mobile <3.2 still expects the v1 shape. Remove once usage hits 0. */
func wrapForV1(r V2Response) V1Response { ... }
```

### Scanning (when the user asks about debt, or CI fails)

Only scan when prompted. Map the user's question to the right command:

| Question | Command |
|---|---|
| What's about to blow up? | `timebombs scan . --within 30d` |
| What has already exploded? | `timebombs scan . --exploded` |
| How much debt total? | `timebombs scan .` |
| What will explode by a given date? | `timebombs scan . --at-time YYYY-MM-DD` |
| Scope to an area. | `timebombs scan ./src/payments` |
| Filter by ticket prefix. | `timebombs scan . --id FLR-` |
| Only what changed in this PR. | `timebombs scan . --changed-only --base origin/main` |
| Machine-readable output. | `timebombs scan . --format json` |
| For code scanning / SARIF upload. | `timebombs scan . --format sarif` |

### CI failure playbook

Exit code `1` from `timebombs scan --max-exploded N` means more than `N` bombs have passed their deadline.

1. `timebombs scan . --exploded` — list the offenders.
2. For each bomb, pick exactly one: **do the work**, or **renegotiate the deadline in a PR**.
3. Never suppress the gate (`|| true`, commenting out, bumping `--max-exploded` without a reason). Hidden debt is worse than visible debt.

### Commit-time checklist

- Marker is exactly `TIMEBOMB(YYYY-MM-DD[, id]): desc` — no typos, no missing colon.
- Deadline is a real, defensible date.
- Description names WHAT, WHY, and what BREAKS.
