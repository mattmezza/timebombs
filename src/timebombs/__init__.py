import logging
from dataclasses import dataclass
from datetime import datetime, timedelta, tzinfo
from functools import partial
from typing import (
    Callable,
    FrozenSet,
    Iterable,
    Optional,
    TypeVar,
    Union,
    cast,
)

import pytz


def now_in(tz: tzinfo) -> datetime:
    return pytz.UTC.localize(datetime.utcnow()).astimezone(tz)


@dataclass(frozen=True, repr=False)
class Timebomb:
    exploding_on: datetime
    arming_on: datetime
    descr: str
    id: Optional[str] = None

    def is_disarmed(self, at: datetime) -> bool:
        return at < self.arming_on

    def is_armed(self, at: datetime) -> bool:
        return self.arming_on <= at < self.exploding_on

    def is_exploded(self, at: datetime) -> bool:
        return self.exploding_on <= at

    def __repr__(self) -> str:
        return "Timebomb({}{}) '{}'.".format(
            f"#{self.id}," if self.id else "",
            self.exploding_on.date(),
            self.descr,
        )


X = TypeVar("X")
Y = TypeVar("Y")
Z = TypeVar("Z")
logger = logging.getLogger(__name__)
log_disarmed = cast(
    Callable[[Timebomb], X], partial(logger.info, "Inactive: %s")
)
log_armed = cast(
    Callable[[Timebomb], Y], partial(logger.warning, "Exploding: %s")
)
log_exploded = cast(
    Callable[[Timebomb], Z], partial(logger.error, "Exploded: %s")
)


class Registry:
    def __init__(self):
        self.bombs: FrozenSet[Timebomb] = frozenset()

    def add(self, b: Timebomb) -> None:
        self.bombs = self.bombs.union({b})

    def bombs_exploded(self, at: datetime) -> Iterable[Timebomb]:
        return (b for b in self.bombs if b.is_exploded(at))

    def bombs_armed(self, at: datetime) -> Iterable[Timebomb]:
        return (b for b in self.bombs if b.is_armed(at))

    def bombs_disarmed(self, at: datetime) -> Iterable[Timebomb]:
        return (b for b in self.bombs if b.is_disarmed(at))


class Boom(Exception):
    def __init__(self, bomb: Timebomb):
        super().__init__(str(bomb))


def timebomb(
    id: Optional[str],
    exploding_on: Union[str, datetime],
    descr: str,
    on_disarmed: Callable[[Timebomb], X] = log_disarmed,
    on_armed: Callable[[Timebomb], Y] = log_armed,
    on_exploded: Callable[[Timebomb], Z] = log_exploded,
    tz: Optional[tzinfo] = None,
    arming_on: Optional[datetime] = None,
    registry: Optional[Registry] = None,
    extreme: bool = False,
) -> Callable[[], Union[X, Y, Z]]:
    if isinstance(exploding_on, str):
        exploding_on = datetime.fromisoformat(exploding_on)

    if arming_on is None:
        arming_on = exploding_on - timedelta(14)

    bomb = Timebomb(exploding_on, arming_on, descr, id)

    if registry:
        registry.add(bomb)

    def tb() -> Union[X, Y, Z]:
        now = datetime.now() if tz is None else now_in(tz)

        if bomb.is_exploded(now):
            if extreme:
                raise Boom(bomb)
            return on_exploded(bomb)
        elif bomb.is_armed(now):
            return on_armed(bomb)
        else:
            return on_disarmed(bomb)

    return tb


__all__ = ["Boom", "Timebomb", "Registry", "timebomb"]
