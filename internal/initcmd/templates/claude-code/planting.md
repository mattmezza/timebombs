---
name: timebombs-planting
description: Plant a TIMEBOMB comment whenever writing code that carries a known deadline to repay — workarounds, backward-compat shims, hardcoded values, naive/unoptimized implementations, feature flags left after rollout, dependency pins, broad error swallowing, or dead code kept alive during a migration. Use this when the code you are about to write is knowingly not where it should end up.
---

# Planting timebombs

The rule: if you write code you *know* is debt, plant a `TIMEBOMB` comment **in the same edit** that introduces the debt. Not later. Not in a separate PR.

## Triggers — when you MUST plant one

IF you are writing any of these, THEN plant a `TIMEBOMB`:

- A workaround or quick fix (demo deadline, fire drill, "just ship it").
- A backward-compat shim, adapter, or wrapper during a migration.
- A hardcoded value that belongs in config (URLs, limits, feature toggles, magic numbers).
- A naive implementation: N+1 queries, missing pagination, no caching, linear scan over something that will grow.
- A feature flag after the experiment has shipped.
- A dependency pin that works around an upstream bug.
- Broad error catching, swallowed errors, `rescue Exception`, `except:` with no type.
- Dead code kept alive "for now" during a transition.

## Non-triggers — DO NOT plant

- Aspirational cleanups ("it would be nice to…").
- Debt with no realistic deadline or external blocker.
- Refactor preferences that aren't actually debt.

For these, use a plain `TODO`. A fake deadline is worse than no deadline.

## Format

```
TIMEBOMB(<YYYY-MM-DD>[, <id>]): <description>
```

- **deadline** — ISO 8601. Required. First arg.
- **id** — ticket (`JIRA-123`, `#317`, `FLR-42`). Optional. Include when a ticket exists.
- **description** — three things, in order:
  1. **WHAT** needs to happen.
  2. **WHY** it was deferred.
  3. **WHAT BREAKS** if it is not done.

Multi-line descriptions: indent continuation lines past the marker (line comments) or keep them inside the same block (`/* */`, `{- -}`).

## Picking the deadline

- Default: **90 days from today** if no better anchor.
- Anchor to a known milestone (sprint end, release date, migration cutover) when one exists.
- Shorter fuse = higher urgency. The deadline IS the severity.

## Before/after examples

### Polling workaround

**Before (debt, no marker):**
```typescript
setInterval(refreshDashboard, 5000);
```

**After (debt, marked):**
```typescript
// TIMEBOMB(2026-07-15, FLR-42): Replace polling with the WS channel.
//   Shipped 5s polling to hit the demo deadline. Under high concurrency
//   this saturates /api/dashboard. Break once FLR-41 (WS auth) lands.
setInterval(refreshDashboard, 5000);
```

### Hardcoded config

**Before:**
```python
RATE_LIMIT = 100
```

**After:**
```python
# TIMEBOMB(2026-06-01): Move rate limit to runtime config.
#   Hardcoded 100 rps because we ship tomorrow. Ops cannot tune without
#   a deploy; will page when an enterprise customer asks for an uplift.
RATE_LIMIT = 100
```

### Backward-compat shim

**Before:**
```go
func wrapForV1(r V2Response) V1Response { ... }
```

**After:**
```go
/* TIMEBOMB(2026-09-30, MIG-17): Drop v1 response wrapper.
   Mobile <3.2 still expects the v1 shape. Analytics show <0.5% of
   requests at the v1 endpoint by 2026-09; remove once that hits 0
   or mobile force-updates, whichever is first. */
func wrapForV1(r V2Response) V1Response { ... }
```

### Feature flag post-rollout

**Before:**
```go
if flags.NewCheckoutEnabled(userID) { newCheckout() } else { oldCheckout() }
```

**After:**
```go
// TIMEBOMB(2026-05-20): Rip out NewCheckoutEnabled branching.
//   Rolled to 100% on 2026-04-20. Keeping the branch hides real call
//   sites and makes the checkout flow harder to read for the next
//   person. Delete the flag, delete oldCheckout.
if flags.NewCheckoutEnabled(userID) { newCheckout() } else { oldCheckout() }
```

## Commit-time checklist

Before committing code with a planted `TIMEBOMB`:

- [ ] Marker is *exactly* `TIMEBOMB(YYYY-MM-DD[, id]): desc` — no typos, no missing colon.
- [ ] Deadline is a real date, in the future, with reasoning you can defend.
- [ ] Description names WHAT, WHY, and what BREAKS.
- [ ] The comment sits **on or directly above** the line it describes.
- [ ] For multi-line descriptions: continuation lines are indented past the marker (line comments) or remain inside the same block (`/* */`).
