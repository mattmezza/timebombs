## Timebombs

`timebombs` is a static-analysis CLI that tracks conscious tech debt with deadlines. A timebomb is a structured comment annotation — no library, no runtime cost.

### Annotation format

```
TIMEBOMB(<YYYY-MM-DD>[, <id>]): <description>
```

- `deadline` — ISO 8601 date. **Required.**
- `id` — ticket reference like `JIRA-123`, `#317`, `FLR-42`. **Optional.**
- `description` — what needs to happen, why it was deferred, what breaks if it's not done.

Continuation lines that are indented past the marker (for line comments) or inside the same block comment (for `/* */`, `{- -}`) are part of the same bomb.

### When to plant a timebomb

Plant one whenever you write code that is *knowingly* not where it should be:

- Workarounds, shortcuts, quick fixes for demos.
- Backward-compatibility shims kept alive during a migration.
- Hardcoded config values (URLs, limits, flags).
- Naive implementations: N+1 queries, missing pagination, missing caching.
- Dependency pins that work around an upstream bug.
- Feature flags left in after a rollout.
- Broad error swallowing, catch-all handlers, silent failures.
- Dead code paths kept alive for a transition.

### When NOT to plant a timebomb

- Aspirational "nice-to-have" comments. Use a plain `TODO`.
- Debt with no realistic deadline or external blocker. A `TODO` is honest; a fake deadline isn't.
- Anything the team hasn't actually agreed is debt.

### Picking a deadline

- Default to **90 days** from today if you have no better anchor.
- Align with a known milestone (sprint end, release, migration cutover).
- Shorter fuse = higher urgency. The deadline is the severity.

### Writing a good description

State three things:

1. **WHAT** needs to happen (the action).
2. **WHY** it was deferred (the context).
3. **WHAT BREAKS** or degrades if it's not done (the consequence).

Multi-line is fine. Be specific. Future-you will thank present-you.

### Examples

```python
# TIMEBOMB(2025-09-01): Remove v1 endpoints after migration complete.
#   The new v2 endpoints are already serving 90% of traffic.
#   Blocked by: mobile app rollout to force-update v1 clients.
```

```typescript
// TIMEBOMB(2025-09-01, JIRA-123): Replace polling with WebSocket.
//   This was a quick fix for the demo. The 5s polling interval
//   puts unnecessary load on the API under high concurrency.
```

```go
/* TIMEBOMB(2025-11-15): Rip out the feature flag scaffolding.
   The experiment shipped; the flag is always-on. Still branching
   everywhere in the checkout flow. */
```

### Running the CLI

Common queries and their flag equivalents:

| Question | Command |
|---|---|
| What's about to blow up? | `timebombs scan . --within 30d` |
| What has exploded? | `timebombs scan . --exploded` |
| How much debt total? | `timebombs scan .` |
| What will explode by a date? | `timebombs scan . --at-time 2026-06-01` |
| Show a specific area's debt. | `timebombs scan ./src/payments` |
| Give me JSON. | `timebombs scan . --format json` |

### Interpreting CI failures

Exit code `1` means a `--max-exploded` threshold was exceeded. Run `timebombs scan . --exploded` to see which bombs triggered the gate.

Two valid resolutions:

1. Do the work — the bomb's deadline is there for a reason.
2. Bump the deadline in a PR. Renegotiating debt openly in a diff is better than letting it rot silently.
