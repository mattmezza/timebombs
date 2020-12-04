What is a timebomb?
===

A timebomb is a modification of the code that is not supposed to last long.
It's a shortcut that we take consciously and that we want to get rid of
after a certain amount of time, or after a certain action has happened.

Think of it as technical debt that you need to clear out.

The goal is to stay clean as you move as fast as you can.

Most important traits of a timebomb
===
- a rationale: the underlying reason for a particular situation
- explosion time: when the timebomb will explode

What happens when a timebomb explodes?
===
Timebombs need to be defused before they explode, if this does not happen
the timebomb itself will activate and will emit an ERROR log message.
In this way we will be notified and we will act upon accordingly.

Prior to bluntly exploding, the timebomb will emit a WARNING log message for a couple of weeks, alerting you thus of the incombent explosion.


Usage:

Define a timebomb at the end of this module like this:

```
timebomb_remove_patch = timebomb("PROJ-1524", "2020-04-01 09:00", "Remove patch.")
```
