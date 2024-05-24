![banner](https://raw.githubusercontent.com/mattmezza/timebombs/master/docs/timebomb.png)

timebombs
===

> Stay clean, move fast.

## intro

Have you ever been in a situation where you found yourself leaving comments in
the code saying something like this:

```
# TODO - remove this after ...
# TODO - this needs cleaning
```

Well, who hasn't!? We all have good intentions to actually go back and do the
removal, do the cleaning etc... But then we:

- forget
- get swamped with other prios
- overtaken by the NMFP (not my f- problem) attitude
- etc.

and these comments end up staying there forever.

## proposal

Wouldn't it be nice to have a mechanism to force teams to actually give a shit?

Let me introduce the concept of a time bomb...

A timebomb is a conscious decision to tackle a certain action at a later
moment in time. It has a detonator that is programmed to explode at some point
in time.

It can stay there, in the code, unarmed and do nothing other than representing
the same thing a comment would have.

When the detonation moment approaches, the time bomb arms itself and you can
make it do things via callback functions. A classic action you might want to
take is to trigger a log message with `WARN` level saying something like:

```
WARN: timebomb PRJ-123 is about to explode in 30 days
```

The above could trigger notification on your alerting system such that work is
properly prioritized and done. At this point, you can either:

- do the work that was post-poned at a later point in time
- or defuse the bomb and post-pone it in the future

> But hey, what's the incentive of actually doing this? What if nothing gets
done anyways and the bomb passes the detonation moment?

Well, if you do nothing, the bomb might actually explode and when that happens
you can make it do something (always via the same mechanism - callback fun).

> What do you mean with *might* explode?

You would be surprised to realize how sometimes old code is
actually dead code. In that case, the time bomb will stay there, inert,
untouched, unharmful (like them WW2 unexploded bombs that people keep finding
here and there throughout Europe).

A typical example of an explosion callback would be emitting a log line, only
this time with an `ERR` log level saying something similar to the above.

This would (or at least *should*) result in some serious alerting going on,
ensuring that:

- the right attention is given to the resolution of the timebomb (i.e. doing
the cleaning, doing the removal, etc.)
- a clear team decision is take to post-pone the timebomb again (making it
visible that debt is being incurred)

For extreme teams, folks who like to live on the edge, upon explosion, a time
bomb could be configured to panic or throw an exception. This definitely makes
things a tag *more* interesting and *encourages* teams to pay back technical
debt.


## show me the code

> All well and swell, but what would it look like to adopt timebombs???

Imagine you have moved on to a `v2` version of your `APIs` and you want to
make sure one day you do go back and clean `v1`. Here's what the hypothetical
code would look like.

```python
# timebombs.py
import functools
import logging
import timebombs


log = logging.getLogger(__name__)
on_armed = lambda tb: log.warning("%s is exploding soon, take action!", tb)
on_exploded = lambda tb: log.error("BOOM! %s exploded!", tb)
timebomb = functools.partial(
    timebombs.timebomb, on_armed=on_armed, on_exploded=on_exploded
)

DEPRECATE_V1_ENDPOINTS = timebomb(
    "JIRA-123",
    "2025-05-22",
    "Endpoints for v1 should be removed by this time.",
)
```

```python
# endpoint.py
from . import timebombs

@get("/v2/resource")
async def get_resource_v2(req, res):
    ...


@get("/v1/resource")
async def get_resource_v1(req, res):
    timebombs.DEPRECATE_V1_ENDPOINTS()
    ...
```

In this example, all timebombs are collected in a single module and imported
where necessary. This is done to have a visual easy way to understand what is
the level of tech debt a team is facing. When the `timebombs.py` module is
getting too big, something's fishy and your life should be miserable anyways.

## i'm not convinced

> This is too much work, I'm not convinced it is more useful than just leaving
comments here and there...

Ok, fair point. What if I told you that you can monitor the amount of unarmed,
armed and exploded timebombs at *ci* time?

In the following example, note the presence of a `timebombs.Registry` to
collect and accumulate all the timebombs you plant around.

```python
# timebombs.py
import functools
import logging
import timebombs


log = logging.getLogger(__name__)
on_armed = lambda tb: log.warning("%s is exploding soon, take action!", tb)
on_exploded = lambda tb: log.error("BOOM! %s exploded!", tb)
reg = timebombs.Registry()
timebomb = functools.partial(
    timebombs.timebomb,
    on_armed=on_armed,
    on_exploded=on_exploded,
    registry=reg,
)

DEPRECATE_V1_ENDPOINTS = timebomb(
    "JIRA-123",
    "2025-05-22",
    "Endpoints for v1 should be removed by this time.",
)
```

Then, somewhere in your *ci* (or from your *cli*):

```bash
$ python -m timebombs "timebombs:reg" --max-armed 20
```

The above would fail with exit status code equal to the number of armed time
bombs (if above `20`) or succeed with exit code `0` otherwise.

It basically asks the question: "As of now, do I have more than `20` armed
timebombs that will explode soon? If so, how many?"

> What if you wanted to see how many time bombs would explode in a month from
now?

```bash
$ python -m timebombs "timebombs:reg" \
    --skip-armed \
    --at-time "$(date -d "$(date +%Y-%m-%d) 1 month" +%Y-%m-%d)"
```

It answers the question: "As of a month from now, will I have any explosion of
a timebomb? If so, how many?"

With this step added to your *ci*, tech debt is actually popping up way
earlier and can be addressed on time.

## summing up

Tech debt is not a problem per se, but **unmanaged** technical debt is.

Timebombs offer *a* possible way to manage it and keep it under control. This
technique offers a way to look into the future (maybe during a cycle of
planning) and answer the question:

> How much time should we reserve for technical debt during the next cycle?

The point is not to avoid debt at any cost, sometimes you just can't. The
whole point is to be honest with yourself.

If, *as a team* you've agreed you were going to pay back technical debt by a
certain time, you should actually do it, right?

Of course, it's going to be difficult to forecast when you will have time to
pay back tech debt. However, the only fact that you actually have to do the
following:

- come up with a point in time by when you want to pay back the debt (forces
you to think of a reasonable time)
- make a *pr* that someone else will review (makes it visible to the rest of
the team - everybody should agree to acquire debt)
- make a conscious decision *as a team* to either pay back the debt (if you
can afford it) or renegotiate it (if you can't)

it pushes teams to spend more time reflecting on these topics, which is good
anyways.

Lastly, there are no guidelines here in terms of what amount of timebombs is
acceptable or not. This is (and will always be) up to you.

### how to get started

```bash
pip install timebombs
```
